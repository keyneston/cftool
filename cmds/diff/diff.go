package diff

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/google/subcommands"
	"github.com/keyneston/cftool/awshelpers"
	"github.com/keyneston/cftool/config"
	"github.com/keyneston/cftool/helpers"
)

const Update = "UPDATE"

var staticCapabilities = []*string{
	aws.String("CAPABILITY_IAM"),
}

type DiffStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB

	Timeout    time.Duration
	PlanOutput string
}

func (*DiffStacks) Name() string { return "diff" }
func (*DiffStacks) Synopsis() string {
	return "Create and print a diff of the live template and the on disk template"
}

func (*DiffStacks) Usage() string {
	return `diff [<filter1>, <filter2>...]
	Gets a description of what would change if the stack updates`
}

func (r *DiffStacks) SetFlags(f *flag.FlagSet) {
	f.DurationVar(&r.Timeout, "t", time.Second*60, "timeout for waiting for results")
	f.StringVar(&r.PlanOutput, "o", "plan.json", "name of file to output the plan ids to")
	f.StringVar(&r.PlanOutput, "plan", "plan.json", "name of file to output the plan ids to")
}

func (r *DiffStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if r.Timeout < time.Second {
		log.Printf("Invalid timeout set: %v", r.Timeout)
		return subcommands.ExitFailure
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	if _, err := os.Stat(r.PlanOutput); err == nil {
		log.Printf("Error: Plan [%q] already exists, exiting", r.PlanOutput)
		return subcommands.ExitFailure
	}

	stacks, err := r.StacksDB.Filter(f.Args()...)
	if err != nil {
		return helpers.ExitErr(err)
	}

	changeSets := []string{}

	for _, s := range stacks.All {
		log.Printf("Diffing: %s", s.Name)
		id, err := r.createChangeSet(s)
		if err != nil {
			return helpers.ExitErr(err)
		}
		if id == "" {
			// If id is empty and there is no error then the there is nothing
			// to change.
			continue
		}

		changeSets = append(changeSets, id)
	}

	if err := r.getResults(ctx, changeSets); err != nil {
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func (r *DiffStacks) getResults(ctx context.Context, ids []string) error {
	errCh := make(chan error, len(ids))
	resultCh := make(chan *cloudformation.DescribeChangeSetOutput, len(ids))
	wg := &sync.WaitGroup{}
	wg.Add(len(ids))

	for _, id := range ids {
		go r.waitForResult(ctx, wg, errCh, resultCh, id)
	}

	wg.Wait()

	close(errCh)
	close(resultCh)

	for err := range errCh {
		log.Printf("Error: %v", err)
	}

	changes := []ChangeEntry{}
	for result := range resultCh {
		changes = append(changes, createChanges(result)...)
	}

	printChanges(changes)
	return nil
}

type logger struct{}

func (logger) Log(stuff ...interface{}) {
	log.Printf("Logging: %#v", stuff)
}

func (r *DiffStacks) waitForResult(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, results chan<- *cloudformation.DescribeChangeSetOutput, id string) {
	defer wg.Done()

	a, err := arn.Parse(id)
	if err != nil {
		errCh <- err
		return
	}

	client := awshelpers.GetCloudFormationClient(a.Region)
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: &id,
	}

	log.Printf("Waiting for changeset to calculate")
	if err := client.WaitUntilChangeSetCreateCompleteWithContext(ctx, input,
		request.WithWaiterDelay(request.ConstantWaiterDelay(time.Second)),
		request.WithWaiterMaxAttempts(int(r.Timeout.Seconds())),
		request.WithWaiterLogger(logger{}),
	); err != nil {
		err = fmt.Errorf("Error fetching changeset (may be due to no changes?): %v", err)
		errCh <- err
		return
	}

	log.Printf("Fetching changeset")
	output, err := client.DescribeChangeSetWithContext(ctx, input)
	if err != nil {
		errCh <- err
		return
	}

	results <- output
}

func (r *DiffStacks) createChangeSet(s *config.StackConfig) (string, error) {
	template, err := s.GetDiskTemplate()
	if err != nil {
		return "", err
	}
	templateHash, err := s.GetDiskTemplateHash()
	if err != nil {
		return "", err
	}
	liveHash, err := s.GetLiveTemplateHash()
	if err != nil {
		return "", err
	}

	// TODO: also check if any parameters change
	if templateHash == liveHash {
		return "", nil
	}

	stackName := s.StackName()
	name, err := changesetName(s)
	if err != nil {
		return "", err
	}

	changeSetInput := &cloudformation.CreateChangeSetInput{
		ChangeSetType: aws.String(Update),
		Capabilities:  staticCapabilities,
		ChangeSetName: &name,
		StackName:     &stackName,
		TemplateBody:  &template,
		Parameters:    s.AWSParams(),
	}
	if err := changeSetInput.Validate(); err != nil {
		return "", err
	}

	region, err := s.Region()
	if err != nil {
		return "", err
	}

	client := awshelpers.GetCloudFormationClient(region)
	res, err := client.CreateChangeSet(changeSetInput)
	if err != nil {
		return "", err
	}

	if res.Id == nil {
		return "", fmt.Errorf("No changeset ID returned from AWS")
	}
	return *res.Id, nil
}

func changesetName(s *config.StackConfig) (string, error) {
	diskHash, err := s.GetDiskTemplateHash()
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%s", s.Name, diskHash)
	// TODO: use regexp to conform to `[a-zA-Z][-a-zA-Z0-9]*`
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "_", "-")

	return name, nil
}
