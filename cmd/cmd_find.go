package cmd

import (
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
var parallelism = 5

func NewFindDuplicatesCommand() *cobra.Command {
	var sourceDir string
	var targetDir string

	c := &cobra.Command{
		Use:   "find",
		Short: "duplicates finds duplicate files",
		RunE:  run,
	}

	c.Flags().StringVarP(&sourceDir, sParam, "s", "", "Source Directory")
	c.Flags().StringVarP(&targetDir, tParam, "t", "", "Target Directory")
	_ = c.MarkFlagRequired(sParam)
	_ = c.MarkFlagRequired(tParam)
	return c
}

type callback func(string)

func run(cmd *cobra.Command, _ []string) error {
	sourceDir, err := cmd.Flags().GetString(sParam)
	if err != nil {
		return err
	}
	targetDir, err := cmd.Flags().GetString(tParam)
	if err != nil {
		return err
	}
	fmt.Printf("Source Directory: '%s'\n", sourceDir)
	fmt.Printf("Target Directory: '%s'\n", targetDir)
	var sourceFiles = make(map[string]*files.File)
	cb := func(path string) {
		file, err := files.NewFile(path)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if file != nil {

		}
		fmt.Printf("File: %s, hash: %s, size: %db \n", path, file.Hash, file.Size)
		sourceFiles[file.FileName()] = file
	}
	getFiles(sourceDir, cb)
	return nil
}
func getFiles(dir string, cb callback) {
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
