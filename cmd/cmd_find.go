package cmd

import (
	"duplicates-github.com/drypa/duplicates-finder/actions"
	"duplicates-github.com/drypa/duplicates-finder/files"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var sParam = "source-dir"
var tParam = "target-dir"
var aParam = "action"
var pParam = "parallelism"

func NewFindDuplicatesCommand() *cobra.Command {
	var sourceDir string
	var targetDir string
	var action string

	c := &cobra.Command{
		Use:   "find",
		Short: "duplicates finds duplicate files",
		RunE:  run,
	}

	c.Flags().StringVarP(&sourceDir, sParam, "s", "", "Source Directory")
	c.Flags().StringVarP(&targetDir, tParam, "t", "", "Target Directory")
	c.Flags().StringVarP(&action, aParam, "a", "Nothing", "Action with duplicates")
	c.Flags().IntP(pParam, "p", 5, "Parallelism")
	_ = c.MarkFlagRequired(sParam)
	_ = c.MarkFlagRequired(tParam)
	return c
}

type callback func(string)

var sourceFiles = make(map[string]*files.File)

func run(cmd *cobra.Command, _ []string) error {
	sourceDir, err := cmd.Flags().GetString(sParam)
	if err != nil {
		return err
	}
	if sourceDir == "" {
		return errors.New("source directory is required")
	}
	targetDir, err := cmd.Flags().GetString(tParam)
	if err != nil {
		return err
	}
	if targetDir == "" {
		return errors.New("target directory is required")
	}
	action, err := cmd.Flags().GetString(aParam)
	if err != nil {
		return err
	}
	a := actions.GetAction(action)

	parallelism, err := cmd.Flags().GetInt(pParam)
	if err != nil {
		return err
	}
	if parallelism <= 1 {
		return errors.New("parallelism must be greater or equal to 1")
	}

	fmt.Printf("Source Directory: '%s'\n", sourceDir)
	fmt.Printf("Target Directory: '%s'\n", targetDir)

	fillSourceFiles(sourceDir, parallelism)
	fmt.Printf("%d files found in source directory\n", len(sourceFiles))
	iterateTargetFiles(targetDir, parallelism, a)

	return nil
}

func iterateTargetFiles(dir string, parallelism int, a actions.Action) {
	filesToDelete := make([]string, 0)
	cb := func(target string) {
		name := filepath.Base(target)
		sourceFile := sourceFiles[name]
		if sourceFile != nil {
			targetFile, err := files.NewFile(target)
			if err == nil {
				if sourceFile.FullPath == targetFile.FullPath {
					return
				}
				if targetFile.Size == sourceFile.Size && targetFile.Hash == sourceFile.Hash {
					switch a {
					case actions.Print:
						fmt.Printf("source %s equals to %s\n", sourceFile.FullPath, targetFile.FullPath)
					case actions.DeleteSource:
						filesToDelete = append(filesToDelete, sourceFile.FullPath)
					case actions.DeleteTarget:
						filesToDelete = append(filesToDelete, targetFile.FullPath)
					default:

					}
				}
			}
		}
	}
	getFiles(dir, cb, parallelism)

	if len(filesToDelete) > 0 {
		for _, fileToDelete := range filesToDelete {
			fmt.Printf("%s\n", fileToDelete)
		}
		fmt.Printf("Do you want to delete these files? (y/n): ")
		a := ""
		fmt.Scan(&a)
		if strings.ToLower(a) == "y" {
			for _, fileToDelete := range filesToDelete {
				deleteFile(fileToDelete)
			}
		}

	}
}

func deleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		fmt.Printf("failed to remove file %s: %v\n", path, err)
	}
	fmt.Printf("%s deleted\n", path)
}

func fillSourceFiles(sourceDir string, parallelism int) {
	var filesChan = make(chan *files.File)
	cb := func(path string) {
		file, err := files.NewFile(path)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if file != nil {
			filesChan <- file
		}
	}
	go func() {
		for file := range filesChan {
			sourceFiles[file.FileName()] = file
		}
	}()
	getFiles(sourceDir, cb, parallelism)

}
func getFiles(dir string, cb callback, parallelism int) {
	res := make(chan string)
	errs := make(chan error)
	semaphore := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	wg.Add(1)
	semaphore <- struct{}{}
	go func() {
		defer wg.Done()
		defer func() { <-semaphore }()
		getFilesFromDirIncludeChildren(dir, res, errs, &wg, semaphore)
	}()

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(res)
		close(errs)
		close(semaphore)
	}(&wg)

	var processWg sync.WaitGroup
	processingSem := make(chan struct{}, parallelism)
	for path := range res {
		processWg.Add(1)
		processingSem <- struct{}{}
		go func(p string) {
			defer processWg.Done()
			defer func() { <-processingSem }()
			cb(p)
		}(path)
	}
	processWg.Wait()

	for err := range errs {
		fmt.Println("Error:", err)
	}
}

func getFilesFromDirIncludeChildren(dir string, res chan<- string, errs chan<- error, wg *sync.WaitGroup, sem chan struct{}) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		errs <- errors.Wrapf(err, "error reading directory %s", dir)
		return
	}
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(path string) {
					defer wg.Done()
					defer func() { <-sem }()
					getFilesFromDirIncludeChildren(path, res, errs, wg, sem)
				}(fullPath)
			default:
				getFilesFromDirIncludeChildren(fullPath, res, errs, wg, sem)
			}
		} else {
			res <- fullPath
		}
	}
}
