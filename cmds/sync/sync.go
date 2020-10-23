package sync

import (
	"context"
	"flag"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/config"
	"golang.org/x/sync/semaphore"
)

const MaxConcurrentAWS = 3

var concurrentAWS = semaphore.NewWeighted(MaxConcurrentAWS)

type SyncStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB

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
	stacks := config.StacksDB{}

	for _, reg := range r.General.Regions {
		regStacks, err := r.getRegion(reg)
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		stacks.AddStack(regStacks...)
	}

	exitCode := subcommands.ExitSuccess
	for _, err := range hydrateStacks(stacks.All) {
		exitCode = subcommands.ExitFailure
		log.Printf("Error: %v", err)
	}

	// Figure out what is new, and what already exists:
	newStacks := []*config.StackConfig{}
	updateStacks := []*config.StackConfig{}

	for _, s := range stacks.All {
		found := r.StacksDB.FindByARN(s.ARN)
		if found == nil {
			newStacks = append(newStacks, s)
		} else {
			updateStacks = append(updateStacks, s)
		}
	}

	log.Printf("%d stacks to update", len(updateStacks))
	log.Printf("%d stacks to create", len(newStacks))

	return exitCode
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

func hydrateStacks(stacks []*config.StackConfig) []error {
	errs := []error{}
	// TODO: parallelise

	wg := &sync.WaitGroup{}
	errsCh := make(chan error, len(stacks))

	for _, s := range stacks {
		wg.Add(1)
		go hydrate(context.TODO(), wg, errsCh, s)
	}

	wg.Wait()
	close(errsCh)

	for err := range errsCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func hydrate(ctx context.Context, wg *sync.WaitGroup, errsCh chan<- error, s *config.StackConfig) {
	defer wg.Done()
	if err := concurrentAWS.Acquire(ctx, 1); err != nil {
		errsCh <- err
		return
	}
	defer concurrentAWS.Release(1)

	if err := s.Hydrate(); err != nil {
		errsCh <- err
		return
	}
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
