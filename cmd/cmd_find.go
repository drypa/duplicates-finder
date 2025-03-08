package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

type runFunc func(cmd *cobra.Command, args []string)

func NewFindDuplicatesCommand() *cobra.Command {
	var sourceDir string
	var targetDir string

	c := &cobra.Command{
		Use:   "find",
		Short: "duplicates finds duplicate files",
		Run:   run(sourceDir, targetDir),
	}
	c.Flags().StringVarP(&sourceDir, "source-dir", "s", "", "Source Directory")
	c.Flags().StringVarP(&targetDir, "target-dir", "t", "", "Target Directory")
	_ = c.MarkFlagRequired("source-dir")
	_ = c.MarkFlagRequired("target-dir")
	return c
}

func run(sourceDir string, targetDir string) runFunc {
	return func(cmd *cobra.Command, args []string) {
		fmt.Printf("Source Directory: %s\n", sourceDir)
		fmt.Printf("Target Directory: %s\n", targetDir)
	}
}
