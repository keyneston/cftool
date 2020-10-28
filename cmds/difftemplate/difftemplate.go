package difftemplate

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/subcommands"
	"github.com/keyneston/cftool/config"
	"github.com/pmezard/go-difflib/difflib"
)

type DiffTemplate struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
	Context  uint
}

func (*DiffTemplate) Name() string { return "diff-template" }
func (*DiffTemplate) Synopsis() string {
	return "Create and print a diff of the live template and the on disk template"
}

func (*DiffTemplate) Usage() string {
	return `diff-template [<filter1>, <filter2>...]
	Provides a unix diff of the live template and the local disk template.`
}

func (r *DiffTemplate) SetFlags(f *flag.FlagSet) {
	f.UintVar(&r.Context, "c", 3, "Number of lines of context; defaults 3")
}

func (r *DiffTemplate) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	stacks, err := r.StacksDB.Filter(f.Args()...)
	if err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	for _, s := range stacks.All {
		log.Printf("Diffing: %s", s.Name)
		if err := r.makeDiff(s); err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}
	}

	return subcommands.ExitSuccess
}

func (r *DiffTemplate) makeDiff(s *config.StackConfig) error {
	diskTemplate, err := s.GetDiskTemplate()
	if err != nil {
		return err
	}

	liveTemplate, err := s.GetLiveTemplate()
	if err != nil {
		return err
	}

	ud := difflib.UnifiedDiff{
		A:        strings.SplitAfter(liveTemplate, "\n"),
		FromFile: filepath.Join("cloudformation", filepath.Base(s.Source)),
		FromDate: "unknown", // TODO: get this date

		B:      strings.SplitAfter(diskTemplate, "\n"),
		ToFile: s.GetDiskTemplateLocation(),
		ToDate: time.Now().Format(time.RFC3339),

		Context: int(r.Context),
		Eol:     "\n",
	}

	if err := difflib.WriteUnifiedDiff(os.Stdout, ud); err != nil {
		return err
	}

	return nil
}
