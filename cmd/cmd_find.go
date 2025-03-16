package cmd

import (
	"duplicates-github.com/drypa/duplicates-finder/actions"
	"duplicates-github.com/drypa/duplicates-finder/files"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
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
	if parallelism <= 0 {
		return errors.New("parallelism must be greater than zero")
	}

	fmt.Printf("Source Directory: '%s'\n", sourceDir)
	fmt.Printf("Target Directory: '%s'\n", targetDir)

	fillSourceFiles(sourceDir, parallelism)
	fmt.Printf("%d files found in source directory\n", len(sourceFiles))
	iterateTargetFiles(targetDir, parallelism, a)
	return nil
}

func iterateTargetFiles(dir string, parallelism int, a actions.Action) {
	cb := func(target string) {
		name := filepath.Base(target)
		file := sourceFiles[name]
		if file != nil {
			sourceFile, err := files.NewFile(target)
			if err == nil {
				if sourceFile.FullPath == file.FullPath {
					fmt.Printf("Same file '%s' skipped\n", sourceFile)
					return
				}
				if file.Size == sourceFile.Size && file.Hash == sourceFile.Hash {
					switch a {
					case actions.Print:
						fmt.Printf("source %s equals to %s\n", file.FullPath, sourceFile)
					case actions.DeleteSource:
						deleteFile(sourceFile.FullPath)
					case actions.DeleteTarget:
						deleteFile(target)
					default:

					}
				}
			}
		}

	}
	getFiles(dir, cb, parallelism)
}

func deleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		fmt.Printf("failed to remove file %s: %v\n", path, err)
	}
	fmt.Printf("%s deleted\n", path)
}

func fillSourceFiles(sourceDir string, parallelism int) {
	cb := func(path string) {
		file, err := files.NewFile(path)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if file != nil {
			sourceFiles[file.FileName()] = file
		}
	}
	getFiles(sourceDir, cb, parallelism)
}
func getFiles(dir string, cb callback, parallelism int) {
	res := make(chan string)
	errs := make(chan error)
	semaphore := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	wg.Add(1)
	semaphore <- struct{}{}
	go getFilesFromDirIncludeChildren(dir, res, errs, &wg, semaphore)

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(res)
		close(errs)
	}(&wg)

	for path := range res {
		cb(path)

	}

	for err := range errs {
		fmt.Println("Error:", err)
	}
}

func getFilesFromDirIncludeChildren(dir string, res chan<- string, errs chan<- error, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	defer func() { <-sem }()
	entries, err := os.ReadDir(dir)
	if err != nil {
		errs <- errors.Wrapf(err, "error reading directory %s", dir)
		return
	}
	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			wg.Add(1)
			sem <- struct{}{}
			getFilesFromDirIncludeChildren(fullPath, res, errs, wg, sem)

		} else {
			res <- fullPath
		}
	}
}
