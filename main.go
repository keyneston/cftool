package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/cmds/configcmd"
	"github.com/keyneston/cfapply/cmds/status"
	"github.com/keyneston/cfapply/config"
)

func registerSubcommands(general *config.GeneralConfig, stacks config.StackSet) {
	// builtin
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	// custom
	subcommands.Register(&status.StatusStacks{stacks}, "")
	subcommands.Register(&sync.SyncStacks{StackSet: stacks, General: general}, "")
	subcommands.Register(&configcmd.PrintConfig{StackSet: stacks, General: general}, "")
}

func main() {
	ctx := context.Background()
	flag.Parse()

	generalConfig, err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		os.Exit(-1)
	}

	stacks, err := config.LoadStacksFromWD()
	if err != nil {
		log.Printf("Error loading stacks: %v", err)
		os.Exit(-1)
	}

	registerSubcommands(generalConfig, stacks)
	os.Exit(int(subcommands.Execute(ctx)))
}
