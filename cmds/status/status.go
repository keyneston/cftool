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

	"github.com/google/subcommands"
	"github.com/keyneston/cfapply/config"
	"github.com/lensesio/tableprinter"
)

type StatusStacks struct {
	StackSet config.StackSet
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

	for _, s := range r.StackSet {
		live, err := s.GetLive()
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		if len(live.Stacks) == 0 {
			log.Printf("Error: got invalid length for %#v", s)
			return subcommands.ExitFailure
		}

		cur := live.Stacks[0]
		region, _ := s.Region()

		template, err := s.GetTemplate()
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}
		liveTemplateHash := HashString(template)
		diskTemplateHash, err := HashFile(s.File)
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}

		entry := StatusEntry{
			Region:              region,
			OurName:             s.Name,
			Name:                *cur.StackName,
			CloudFormationDrift: "unknown",
			TemplateDiff:        liveTemplateHash != diskTemplateHash,
		}

		if cur.DriftInformation != nil && cur.DriftInformation.StackDriftStatus != nil {
			entry.CloudFormationDrift = *cur.DriftInformation.StackDriftStatus
		}

		entries = append(entries, entry)
	}

	tableprinter.Print(os.Stdout, entries)
	return subcommands.ExitSuccess
}

type StatusEntry struct {
	Region              string `header:"aws region"`
	Name                string `header:"stackname"`
	OurName             string `header:"internal name"`
	CloudFormationDrift string `header:"cloudformation drift"`
	TemplateDiff        bool   `header:"template drift"`
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
