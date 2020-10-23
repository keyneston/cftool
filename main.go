package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/cmds/configcmd"
	"github.com/keyneston/cfapply/cmds/status"
	"github.com/keyneston/cfapply/cmds/sync"
	"github.com/keyneston/cfapply/config"
)

func registerSubcommands(general *config.GeneralConfig, stacks *config.StacksDB) {
	// builtin
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	// custom
	subcommands.Register(&status.StatusStacks{StacksDB: stacks, General: general}, "")
	subcommands.Register(&sync.SyncStacks{StacksDB: stacks, General: general}, "")
	subcommands.Register(&configcmd.PrintConfig{StacksDB: stacks, General: general}, "")
}

func main() {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())
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
