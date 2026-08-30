package main

import (
	"context"
	"duplicates-github.com/drypa/duplicates-finder/cmd"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command := cmd.NewFindDuplicatesCommand()
	command.SetContext(ctx)
	if err := command.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		return
	}
}
