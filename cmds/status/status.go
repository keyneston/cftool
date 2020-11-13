package status

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/subcommands"
	"github.com/keyneston/cftool/awshelpers"
	"github.com/keyneston/cftool/config"
	"github.com/lensesio/tableprinter"
)

type StatusStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
}

func (*StatusStacks) Name() string     { return "status" }
func (*StatusStacks) Synopsis() string { return "Lists the stacks and their status" }
func (*StatusStacks) Usage() string {
	return `status [<filter1>, <filter2>...]
	Lists the stacks and their status. Filters are additive.
`
}

func (r *StatusStacks) SetFlags(f *flag.FlagSet) {
}

func (r *StatusStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	r.General.Log.Debug("Starting StatusStacks.Execute()")

	entries := []StatusEntry{}
	errors := []error{}

	wg := &sync.WaitGroup{}

	stacks, err := r.StacksDB.Filter(f.Args()...)
	if err != nil {
		r.General.Log.Errorf("%v", err)
		return subcommands.ExitFailure
	}
	r.General.Log.Debug("debug: got statcks %#v", stacks)

	results := make(chan StatusEntry, r.StacksDB.Len())
	errCh := make(chan error, r.StacksDB.Len())
	wg.Add(stacks.Len())
	for _, s := range stacks.All {
		go r.getEntry(wg, results, errCh, s)
	}

	wg.Wait()
	close(results)
	close(errCh)

	for entry := range results {
		entries = append(entries, entry)
	}

	for err := range errCh {
		r.General.Log.Errorf("%v", err)
		errors = append(errors, err)
	}

	tableprinter.Print(os.Stdout, entries)

	if len(errors) != 0 {
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (r *StatusStacks) getEntry(wg *sync.WaitGroup, results chan<- StatusEntry, errors chan<- error, s *config.StackConfig) {
	defer wg.Done()

	region, _ := s.Region()
	awshelpers.Ratelimit(context.TODO(), region, func() {
		live, err := s.GetLive()
		if err != nil {
			r.General.Log.Errorf("%v", err)
			errors <- err
			return
		}

		if len(live.Stacks) == 0 {
			errors <- fmt.Errorf("got invalid length for %#v", s)
			return
		}

		cur := live.Stacks[0]
		region, _ := s.Region()

		entry := StatusEntry{
			Region:              region,
			OurName:             s.Name,
			Name:                *cur.StackName,
			CloudFormationDrift: "unknown",
		}

		if s.File != "" {
			liveTemplateHash, err := s.GetLiveTemplateHash()
			if err != nil {
				errors <- err
				return
			}
			diskTemplateHash, err := s.GetDiskTemplateHash()
			if err != nil {
				errors <- err
				return
			}

			entry.TemplateDiff = aws.Bool(liveTemplateHash != diskTemplateHash)
		}

		if cur.DriftInformation != nil && cur.DriftInformation.StackDriftStatus != nil {
			entry.CloudFormationDrift = *cur.DriftInformation.StackDriftStatus
		}

		results <- entry
	})
}

type StatusEntry struct {
	Region              string `header:"aws region"`
	Name                string `header:"stackname"`
	OurName             string `header:"internal name"`
	CloudFormationDrift string `header:"cloudformation drift"`
	TemplateDiff        *bool  `header:"template drift"`
}
