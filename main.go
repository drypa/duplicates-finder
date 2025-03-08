package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func main() {
	var sourceDir string
	var targetDir string
	command := cobra.Command{
		Use:   "duplicates",
		Short: "duplicates finds duplicate files",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Source Directory: %s\n", sourceDir)
			fmt.Printf("Target Directory: %s\n", targetDir)
		},
	}
	command.Flags().StringVarP(&sourceDir, "source-dir", "s", "", "Source Directory")
	command.Flags().StringVarP(&targetDir, "target-dir", "t", "", "Target Directory")
	_ = command.MarkFlagRequired("source-dir")
	_ = command.MarkFlagRequired("target-dir")
	if err := command.Execute(); err != nil {
		fmt.Println(err)
		return
	}
}
