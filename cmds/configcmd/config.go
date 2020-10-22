package configcmd

import (
	"context"
	"flag"
	"io"
	"os"

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/config"
	"github.com/keyneston/tabslib"
)

type PrintConfig struct {
	General  *config.GeneralConfig `json:"general"`
	StackSet config.StackSet       `json:"stacks"`
}

func (*PrintConfig) Name() string     { return "config" }
func (*PrintConfig) Synopsis() string { return "Print a copy of the config" }
func (*PrintConfig) Usage() string {
	return `config:
	Print a copy of the config
`
}

func (r *PrintConfig) SetFlags(f *flag.FlagSet) {
}

func (r *PrintConfig) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	io.WriteString(os.Stdout, tabslib.PrettyString(r))
	io.WriteString(os.Stdout, "\n")
	return subcommands.ExitSuccess
}
