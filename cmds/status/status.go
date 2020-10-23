package status

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/awshelpers"
	"github.com/keyneston/cfapply/config"
	"github.com/lensesio/tableprinter"
)

type StatusStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
}

func (*StatusStacks) Name() string     { return "status" }
func (*StatusStacks) Synopsis() string { return "Lists the stacks and their status" }
func (*StatusStacks) Usage() string {
	return `status:
	Lists the stacks and their status
`
}

func (r *StatusStacks) SetFlags(f *flag.FlagSet) {
}

func (r *StatusStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	entries := []StatusEntry{}
	errors := []error{}

	results := make(chan StatusEntry, r.StacksDB.Len())
	errCh := make(chan error, r.StacksDB.Len())
	wg := &sync.WaitGroup{}
	wg.Add(r.StacksDB.Len())

	for _, s := range r.StacksDB.All {
		go r.getEntry(wg, results, errCh, s)
	}

	wg.Wait()
	close(results)
	close(errCh)

	for entry := range results {
		entries = append(entries, entry)
	}

	for err := range errCh {
		log.Printf("Error: %v", err)
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
			log.Printf("Error: %v", err)
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
			template, err := s.GetTemplate()
			if err != nil {
				errors <- err
				return
			}
			liveTemplateHash := HashString(template)
			diskTemplateHash, err := HashFile(s.File)
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

func HashFile(filename string) (string, error) {
	hasher := sha256.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("error hashing %q: %v", filename, err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("error hashing %q: %v", filename, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func HashString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))

	return hex.EncodeToString(hasher.Sum(nil))
}
