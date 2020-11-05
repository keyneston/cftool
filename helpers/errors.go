package helpers

import (
	"fmt"
	"log"

	"github.com/google/subcommands"
)

// ExitFailure logs a message and returns the subcommands.ExitFailure. It is a wrapper to prevent writing the following all over the place:
//
//     log.Printf("Error: %v", err)
//     return subcommands.ExitFailure
//
func Exitf(msg string, items ...interface{}) subcommands.ExitStatus {
	log.Printf("Error: %v", fmt.Sprintf(msg, items...))
	return subcommands.ExitFailure
}

func ExitErr(err error) subcommands.ExitStatus {
	log.Printf("Error: %v", err)
	return subcommands.ExitFailure
}
