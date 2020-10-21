package list

import (
	"context"
	"flag"
	"log"

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/config"
)

type ListStacks struct {
	StackSet config.StackSet
}

func (*ListStacks) Name() string     { return "status" }
func (*ListStacks) Synopsis() string { return "List the stacks and their status" }
func (*ListStacks) Usage() string {
	return `status:
	Lists the stacks and their status
`
}

func (r *ListStacks) SetFlags(f *flag.FlagSet) {
}

func (r *ListStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	client, err := config.AWSClient()
	if err != nil {
		log.Printf("Error making aws client: %q", err)
		return subcommands.ExitFailure
	}

	for _, s := range r.StackSet {
		// FIXME: This is not thread safe
		live, err := s.GetLive(client)
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		log.Printf("%#v", live)
	}

	return subcommands.ExitSuccess
}
