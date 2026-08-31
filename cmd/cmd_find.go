package cmd

import (
	"context"
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
	ctx := cmd.Context()
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

	if err := ctx.Err(); err != nil {
		fmt.Println("Operation cancelled")
		return nil
	}

	fmt.Println("Indexing source directory...")
	if err := fillSourceFiles(ctx, sourceDir, parallelism); err != nil {
		if ctx.Err() != nil {
			fmt.Println("Operation cancelled during indexing")
			return nil
		}
		return err
	}
	fmt.Printf("%d files found in source directory\n", len(sourceFiles))

	if err := ctx.Err(); err != nil {
		fmt.Println("Operation cancelled")
		return nil
	}

	fmt.Println("Iterate target directory...")
	iterateTargetFiles(ctx, targetDir, parallelism, a)

	if ctx.Err() != nil {
		fmt.Println("Operation cancelled")
		return nil
	}

	return nil
}

func iterateTargetFiles(ctx context.Context, dir string, parallelism int, a actions.Action) {
	filesToDeleteSet := make(map[string]struct{})
	var mu sync.Mutex
	cb := func(target string) {
		name := filepath.Base(target)
		sourceFile := sourceFiles[name]
		if sourceFile != nil {
			targetFile, err := files.NewFile(ctx, target)
			if err == nil {
				if sourceFile.FullPath == targetFile.FullPath {
					return
				}
				if targetFile.Size == sourceFile.Size && targetFile.Hash == sourceFile.Hash {
					switch a {
					case actions.Print:
						fmt.Printf("source %s equals to %s\n", sourceFile.FullPath, targetFile.FullPath)
					case actions.DeleteSource:
						mu.Lock()
						filesToDeleteSet[sourceFile.FullPath] = struct{}{}
						mu.Unlock()
					case actions.DeleteTarget:
						mu.Lock()
						filesToDeleteSet[targetFile.FullPath] = struct{}{}
						mu.Unlock()
					default:

					}
				}
			}
		}
	}
	getFiles(ctx, dir, cb, parallelism)

	if ctx.Err() != nil {
		return
	}

	if len(filesToDeleteSet) > 0 {
		filesToDelete := make([]string, 0, len(filesToDeleteSet))
		for f := range filesToDeleteSet {
			filesToDelete = append(filesToDelete, f)
		}
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

func fillSourceFiles(ctx context.Context, sourceDir string, parallelism int) error {
	sourceFiles = make(map[string]*files.File)
	var filesChan = make(chan *files.File)
	cb := func(path string) {
		if err := ctx.Err(); err != nil {
			return
		}
		file, err := files.NewFile(ctx, path)
		if err != nil {
			if files.IsCancelled(err) {
				return
			}
			fmt.Println("Error:", err)
			return
		}
		filesChan <- file
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for file := range filesChan {
			sourceFiles[file.FileName()] = file
		}
	}()
	err := getFiles(ctx, sourceDir, cb, parallelism)
	close(filesChan)
	wg.Wait()
	return err
}
func getFiles(ctx context.Context, dir string, cb callback, parallelism int) error {
	res := make(chan string)
	errs := make(chan error)
	semaphore := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	wg.Add(1)
	semaphore <- struct{}{}
	go func() {
		defer wg.Done()
		defer func() { <-semaphore }()
		getFilesFromDirIncludeChildren(ctx, dir, res, errs, &wg, semaphore)
	}()

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(res)
		close(errs)
		close(semaphore)
	}(&wg)

	var processWg sync.WaitGroup
	processingSem := make(chan struct{}, parallelism)
	for {
		select {
		case <-ctx.Done():
			goto done
		case path, ok := <-res:
			if !ok {
				processWg.Wait()
				goto done
			}
			processWg.Add(1)
			processingSem <- struct{}{}
			go func(p string) {
				defer processWg.Done()
				defer func() { <-processingSem }()
				cb(p)
			}(path)
		}
	}

done:
	errCh := make(chan error, 1)
	go func() {
		var firstErr error
		for err := range errs {
			if firstErr == nil {
				firstErr = err
			}
			fmt.Println("Error:", err)
		}
		errCh <- firstErr
	}()
	processWg.Wait()
	<-errCh
	return nil
}

func getFilesFromDirIncludeChildren(ctx context.Context, dir string, res chan<- string, errs chan<- error, wg *sync.WaitGroup, sem chan struct{}) {
	if err := ctx.Err(); err != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		errs <- errors.Wrapf(err, "error reading directory %s", dir)
		return
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return
		}
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(path string) {
					defer wg.Done()
					defer func() { <-sem }()
					getFilesFromDirIncludeChildren(ctx, path, res, errs, wg, sem)
				}(fullPath)
			default:
				getFilesFromDirIncludeChildren(ctx, fullPath, res, errs, wg, sem)
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case res <- fullPath:
			}
		}
	}
}
