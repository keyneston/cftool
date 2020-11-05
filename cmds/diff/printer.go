package diff

import (
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/fatih/color"
	"github.com/lensesio/tableprinter"
)

type ChangeEntry struct {
	Stack  string `header:"Stack"`
	Action string `header:"Action"`
	Type   string `header:"Type"`
	Name   string `header:"Resource Name"`
}

var (
	red   = color.New(color.FgRed).SprintFunc()
	blue  = color.New(color.FgBlue).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

func printChanges(changes []ChangeEntry) {
	sort.Slice(changes, func(i, j int) bool {
		return (changes[j].Stack > changes[i].Stack) || (changes[j].Name > changes[i].Name)
	})

	for i, change := range changes {
		switch change.Action {
		case "Remove":
			changes[i].Action = red(change.Action)
		case "Modify":
			changes[i].Action = blue(change.Action)
		case "Add":
			changes[i].Action = green(change.Action)
		}
	}

	printer := tableprinter.New(os.Stdout)

	printer.Print(changes)
}

func createChanges(out *cloudformation.DescribeChangeSetOutput) []ChangeEntry {
	entries := []ChangeEntry{}

	for _, change := range out.Changes {
		var entry ChangeEntry

		if change.Type == nil {
			continue
		}

		switch *change.Type {
		case "Resource":
			entry = fromResourceChange(out, change)
		}

		entries = append(entries, entry)
	}

	return entries
}

func fromResourceChange(out *cloudformation.DescribeChangeSetOutput, c *cloudformation.Change) ChangeEntry {
	entry := ChangeEntry{}

	entry.Stack = getAWSString(out.StackName)
	entry.Action = getAWSString(c.ResourceChange.Action)
	entry.Name = getAWSString(c.ResourceChange.LogicalResourceId)
	entry.Type = getAWSString(c.ResourceChange.ResourceType)

	return entry
}

func getAWSString(in *string) string {
	if in == nil {
		return ""
	}

	return *in
}
