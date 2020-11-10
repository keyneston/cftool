package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/subcommands"
	"github.com/keyneston/cftool/cmds/configcmd"
	"github.com/keyneston/cftool/cmds/diff"
	"github.com/keyneston/cftool/cmds/difftemplate"
	"github.com/keyneston/cftool/cmds/fetch"
	"github.com/keyneston/cftool/cmds/sshcmd"
	"github.com/keyneston/cftool/cmds/status"
	"github.com/keyneston/cftool/config"
)

func registerSubcommands(general *config.GeneralConfig, stacks *config.StacksDB) {
	// builtin
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	// custom
	subcommands.Register(&status.StatusStacks{StacksDB: stacks, General: general}, "")
	subcommands.Register(&fetch.FetchStacks{StacksDB: stacks, General: general}, "")
	subcommands.Register(&configcmd.PrintConfig{StacksDB: stacks, General: general}, "")
	subcommands.Register(&diff.DiffStacks{StacksDB: stacks, General: general}, "")
	subcommands.Register(&difftemplate.DiffTemplate{StacksDB: stacks, General: general}, "")
	subcommands.Register(&sshcmd.SSHcmd{StacksDB: stacks, General: general}, "")
}

func main() {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	shouldDebug := false

	flag.BoolVar(&shouldDebug, "debug", shouldDebug, "enable debug output")
	flag.Parse()

	generalConfig, err := config.LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		os.Exit(-1)
	}

	generalConfig.ShouldDebug = shouldDebug

	stacks, err := generalConfig.LoadStacks()
	if err != nil {
		log.Printf("Error loading stacks: %v", err)
		os.Exit(-1)
	}

	registerSubcommands(generalConfig, stacks)
	os.Exit(int(subcommands.Execute(ctx)))
}
