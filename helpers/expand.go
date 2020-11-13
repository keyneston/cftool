package helpers

import (
	"log"

	"github.com/mitchellh/go-homedir"
)

// Expand takes a string and attempts to expand the any references to a homedir
// in the string. If it fails, it just returns the input.
func Expand(in string) string {
	if long, err := homedir.Expand(in); err != nil {
		log.Printf("Warning: Error expanding homedir: %v", err)
	} else {
		return long
	}

	return in
}
