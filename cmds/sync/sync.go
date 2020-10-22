package sync

import (
	"context"
	"flag"
	"log"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/config"
	"github.com/keyneston/tabslib"
)

type SyncStacks struct {
	General  *config.GeneralConfig
	StackSet config.StackSet

	Noop bool
}

func (*SyncStacks) Name() string     { return "sync" }
func (*SyncStacks) Synopsis() string { return "Sync the stacks and their parameters" }
func (*SyncStacks) Usage() string {
	return `sync:
	Syncs the stacks and their parameters
`
}

func (r *SyncStacks) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&r.Noop, "noop", false, "noop don't write changes")
}

func (r *SyncStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Printf("%s", tabslib.PrettyString(r))
	stacks := []*config.StackConfig{}

	for _, reg := range r.General.Regions {
		regStacks, err := r.getRegion(reg)
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		stacks = append(stacks, regStacks...)
	}

	log.Printf("Got: %v", tabslib.PrettyString(stacks))
	log.Printf("%d stacks", len(stacks))

	return subcommands.ExitSuccess
}

func (r *SyncStacks) getRegion(region string) ([]*config.StackConfig, error) {
	log.Printf("INFO: Fetching %q", region) // TODO: switch to proper logger

	client, err := config.AWSClient(region)
	if err != nil {
		return nil, err
	}

	stacks := []*config.StackConfig{}
	var next *string

	for {
		res, err := client.ListStacks(&cloudformation.ListStacksInput{
			NextToken: next,
		})
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, convertToLocal(res.StackSummaries)...)

		if res.NextToken == nil {
			break
		}
		next = res.NextToken

	}

	return stacks, nil
}

func convertToLocal(stacks []*cloudformation.StackSummary) []*config.StackConfig {
	res := []*config.StackConfig{}

	for _, s := range stacks {
		res = append(res, &config.StackConfig{
			Name: "unknown",
			ARN:  *s.StackId,
		})
	}
	return res
}
