package main

import (
	"duplicates-github.com/drypa/duplicates-finder/cmd"
	"fmt"
)

func main() {
	command := cmd.NewFindDuplicatesCommand()
	if err := command.Execute(); err != nil {
		fmt.Println(err)
		return
	}
}
