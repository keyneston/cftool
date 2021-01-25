package show

import (
	"context"
	"flag"

	"github.com/google/subcommands"
	"github.com/keyneston/cftool/config"
)

type ShowStack struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
}

func (*ShowStack) Name() string     { return "show" }
func (*ShowStack) Synopsis() string { return "Shows a set of stacks, and their parameters" }
func (*ShowStack) Usage() string {
	return `show [<filter1>, <filter2>...]
`
}

func (r *ShowStack) SetFlags(f *flag.FlagSet) {
}

func (r *ShowStack) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	r.General.Log.Debug("Starting ShowStack.Execute()")

	// 	entries := []StatusEntry{}
	// 	errors := []error{}
	//
	// 	wg := &sync.WaitGroup{}
	//
	// 	stacks, err := r.StacksDB.Filter(f.Args()...)
	// 	if err != nil {
	// 		r.General.Log.Errorf("%v", err)
	// 		return subcommands.ExitFailure
	// 	}
	// 	r.General.Log.Debug("debug: got statcks %#v", stacks)
	//
	// 	results := make(chan StatusEntry, r.StacksDB.Len())
	// 	errCh := make(chan error, r.StacksDB.Len())
	// 	wg.Add(stacks.Len())
	// 	for _, s := range stacks.All {
	// 		go r.getEntry(wg, results, errCh, s)
	// 	}
	//
	// 	wg.Wait()
	// 	close(results)
	// 	close(errCh)
	//
	// 	for entry := range results {
	// 		entries = append(entries, entry)
	// 	}
	//
	// 	for err := range errCh {
	// 		r.General.Log.Errorf("%v", err)
	// 		errors = append(errors, err)
	// 	}
	//
	// 	tableprinter.Print(os.Stdout, entries)
	//
	// 	if len(errors) != 0 {
	// 		return subcommands.ExitFailure
	// 	}
	//
	return subcommands.ExitSuccess
}
