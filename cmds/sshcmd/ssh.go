package sshcmd

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/google/subcommands"
	"github.com/keyneston/cftool/config"
)

type SSHcmd struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB
}

func (*SSHcmd) Name() string { return "ssh" }
func (*SSHcmd) Synopsis() string {
	return "SSH into a host from a stack"
}

func (*SSHcmd) Usage() string {
	return `ssh [<filter1>, <filter2>...] [-- commands to ssh]
	Grab a host from a stack and ssh into it

	If a -- is given all additional flags will be passed to ssh.
	e.g.


	"cftool ssh myStack -- -v" => "ssh -v 192.0.2.0"

	`
}

func (r *SSHcmd) SetFlags(f *flag.FlagSet) {
}

func (r *SSHcmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	filters := []string{}
	additionalSSHArgs := []string{}
	args := f.Args()
	for i, arg := range args {
		if arg != "--" {
			continue
		}

		filters = args[0:i]
		if len(args) > i {
			additionalSSHArgs = args[i+1 : len(args)]
		}

	}

	stacks, err := r.StacksDB.Filter(filters...)
	if err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	allServers := []string{}
	for _, s := range stacks.All {
		servers, err := s.FetchServers()
		if err != nil {
			log.Printf("Error: %v", err)
			return subcommands.ExitFailure
		}
		allServers = append(allServers, servers...)
	}

	log.Printf("Servers: %v", allServers)
	if len(allServers) == 0 {
		log.Printf("Can't find server")
		return subcommands.ExitFailure
	}

	bin := "/usr/bin/ssh"
	sshArgs := []string{bin}
	sshArgs = append(sshArgs, additionalSSHArgs...)
	sshArgs = append(sshArgs, allServers[0])
	log.Printf("exec %v", strings.Join(sshArgs, " "))

	if err := syscall.Exec(bin, sshArgs, os.Environ()); err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
