package fetch

import (
	"context"
	"flag"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/google/subcommands"
	"github.com/keyneston/cftool/awshelpers"
	"github.com/keyneston/cftool/config"
	"golang.org/x/sync/semaphore"
)

const MaxConcurrentAWS = 3

var concurrentAWS = semaphore.NewWeighted(MaxConcurrentAWS)

type FetchStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB

	Noop bool
}

func (*FetchStacks) Name() string     { return "fetch" }
func (*FetchStacks) Synopsis() string { return "Fetch the stacks and their parameters" }
func (*FetchStacks) Usage() string {
	return `fetch:
	Fetches the stacks and their parameters
`
}

func (r *FetchStacks) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&r.Noop, "noop", false, "noop don't write changes")
}

func (r *FetchStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fetchedStacks := config.StacksDB{}

	for _, reg := range r.General.Regions {
		regStacks, err := r.getRegion(reg)
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		fetchedStacks.AddStack(regStacks...)
	}

	filteredDiskStacks, err := r.StacksDB.Filter(f.Args()...)
	if err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	filteredFetchedStacks, err := fetchedStacks.Filter(f.Args()...)
	if err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	// Figure out what is new, and what already exists:
	newStacks := []*config.StackConfig{}
	updateStacks := []*config.StackConfig{}

	for _, s := range fetchedStacks.All {
		filtered := filteredFetchedStacks.FindByARN(s.ARN)
		onDisk := filteredDiskStacks.FindByARN(s.ARN)

		// if filtered and exists: update
		// if filtered and not exists: create
		if filtered != nil && onDisk == nil {
			newStacks = append(newStacks, filtered)
		} else if onDisk != nil {
			updateStacks = append(updateStacks, onDisk) // pass the onDisk version since it has some data we want (Source)
		}
	}

	exitCode := subcommands.ExitSuccess
	if err := r.updateStacks(updateStacks); err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	if err := r.createStacks(newStacks); err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	return exitCode
}

func (r *FetchStacks) updateStacks(stacks []*config.StackConfig) error {
	log.Printf("INFO: updating %d stacks", len(stacks))
	for _, err := range hydrateStacks(stacks) {
		if err != nil {
			return err
		}
	}

	for _, s := range stacks {
		location := s.Location()

		disk, err := config.LoadStackFromFile(location)
		if err != nil {
			return err
		}

		disk = s

		if err := disk.Save(location); err != nil {
			return err
		}
	}

	return nil
}

func (r *FetchStacks) createStacks(stacks []*config.StackConfig) error {
	log.Printf("INFO: creating %d stacks", len(stacks))
	for _, err := range hydrateStacks(stacks) {
		if err != nil {
			return err
		}
	}

	for _, s := range stacks {
		location := s.Location()

		if err := s.Save(location); err != nil {
			return err
		}
	}

	return nil
}

func (r *FetchStacks) getRegion(region string) ([]*config.StackConfig, error) {
	log.Printf("INFO: Fetching %q", region) // TODO: switch to proper logger

	client := awshelpers.GetCloudFormationClient(region)

	stacks := []*config.StackConfig{}

	input := &cloudformation.ListStacksInput{}
	if err := client.ListStacksPagesWithContext(
		context.TODO(),
		input,
		func(res *cloudformation.ListStacksOutput, more bool) bool {
			stacks = append(stacks, convertToLocal(res.StackSummaries)...)

			return more
		}); err != nil {
		return nil, err
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

	region, _ := s.Region()
	awshelpers.Ratelimit(ctx, region, func() {
		if err := s.Hydrate(); err != nil {
			errsCh <- err
			return
		}
	})
}

func convertToLocal(stacks []*cloudformation.StackSummary) []*config.StackConfig {
	res := []*config.StackConfig{}

	for _, s := range stacks {
		res = append(res, &config.StackConfig{
			Name: *s.StackName,
			ARN:  *s.StackId,
		})
	}
	return res
}
