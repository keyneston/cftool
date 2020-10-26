package diff

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/google/subcommands"
	"github.com/keyneston/cftool/config"
	"github.com/keyneston/tabslib"
)

const Update = "UPDATE"

var staticCapabilities = []*string{
	aws.String("CAPABILITY_IAM"),
}

type DiffStacks struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
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
}

func (r *DiffStacks) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	stacks, err := r.StacksDB.Filter(f.Args()...)
	if err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	for _, s := range stacks.All {
		log.Printf("Diffing: %s", s.Name)
		if err := r.createChangeSet(s); err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}

func (r *DiffStacks) createChangeSet(s *config.StackConfig) error {
	template, err := s.GetDiskTemplate()
	if err != nil {
		return err
	}

	stackName := s.StackName()
	name, err := changesetName(s)
	if err != nil {
		return err
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
		return err
	}
	log.Printf("Sending: %s", tabslib.PrettyString(changeSetInput))

	//region, err := s.Region()
	//if err != nil {
	//	return err
	//}

	//client := awshelpers.GetClient(region)
	//res, err := client.CreateChangeSet(changeSetInput)
	//if err != nil {
	//	return err
	//}
	//log.Printf("Res: %s", tabslib.PrettyString(res))

	return nil
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
