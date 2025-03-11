package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"sync"
)

func NewFindDuplicatesCommand() *cobra.Command {
	var sourceDir string
	var targetDir string

	c := &cobra.Command{
		Use:   "find",
		Short: "duplicates finds duplicate files",
		RunE:  run,
	}
	c.Flags().StringVarP(&sourceDir, "source-dir", "s", "", "Source Directory")
	c.Flags().StringVarP(&targetDir, "target-dir", "t", "", "Target Directory")
	_ = c.MarkFlagRequired("source-dir")
	_ = c.MarkFlagRequired("target-dir")
	return c
}

func run(cmd *cobra.Command, _ []string) error {
	sourceDir, err := cmd.Flags().GetString("source-dir")
	if err != nil {
		return err
	}
	targetDir, err := cmd.Flags().GetString("target-dir")
	if err != nil {
		return err
	}
	fmt.Printf("Source Directory: '%s'\n", sourceDir)
	fmt.Printf("Target Directory: '%s'\n", targetDir)
	getFiles(sourceDir)
	return nil
}
func getFiles(dir string) {
	res := make(chan string)
	errs := make(chan error)
	semaphore := make(chan struct{}, 5)
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
		fmt.Println("File:", path)
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
