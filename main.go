package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/cmds/status"
	"github.com/keyneston/cfapply/config"
)

func registerSubcommands(stacks config.StackSet) {
	// builtin
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	// custom
	subcommands.Register(&status.StatusStacks{stacks}, "")
}

func main() {
	ctx := context.Background()
	flag.Parse()

	stacks, err := config.LoadStacksFromWD()
	if err != nil {
		log.Printf("Error loading stacks: %v", err)
		os.Exit(-1)
	}

	registerSubcommands(stacks)
	os.Exit(int(subcommands.Execute(ctx)))
}
