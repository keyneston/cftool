package sshcmd

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/google/subcommands"
	"github.com/keyneston/cftool/config"
)

type SSHcmd struct {
	General  *config.GeneralConfig
	StacksDB *config.StacksDB

	ServerOffset uint
	RandomServer bool
	Pdsh         bool
	Noop         bool
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
	f.BoolVar(&r.RandomServer, "r", false, "Select a server at random")
	f.UintVar(&r.ServerOffset, "o", 0, "Pick server N")
	f.BoolVar(&r.Pdsh, "p", false, "Run with PDSH for parallel")
	f.BoolVar(&r.Noop, "n", false, "Don't actually execute a program")
}

func (r *SSHcmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	additionalSSHArgs := []string{}
	args := f.Args()
	filters := args
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
		r.General.Log.Errorf("%v", err)
		return subcommands.ExitFailure
	}

	allServers := []string{}
	for _, s := range stacks.All {
		for _, server := range s.Servers {
			allServers = append(allServers, server.PrivateIP)
		}
	}

	r.General.Log.Debugf("Servers: %v", allServers)
	if len(allServers) == 0 {
		r.General.Log.Errorf("Can't find server")
		return subcommands.ExitFailure
	}

	if r.RandomServer {
		r.ServerOffset = uint(rand.Uint64())
	}
	offset := r.ServerOffset % uint(len(allServers))

	r.General.Log.Debug("Picking server %d", offset)
	server := allServers[offset]

	if r.Pdsh {
		err = r.ExecPDSH(allServers, additionalSSHArgs)
	} else {
		err = r.ExecSSH(server, additionalSSHArgs)
	}

	if err != nil {
		r.General.Log.Errorf("%v", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (r SSHcmd) Exec(command string, args []string) error {
	bin, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("Can't find binary %q: %v", command, err)
	}

	sshArgs := []string{bin}
	sshArgs = append(sshArgs, args...)
	r.General.Log.Debugf("exec %v", strings.Join(sshArgs, " "))

	if r.Noop {
		fmt.Printf("exec %v", strings.Join(sshArgs, " "))
		return nil
	}

	if err := syscall.Exec(bin, sshArgs, os.Environ()); err != nil {
		return err
	}

	return nil
}

func (r SSHcmd) ExecPDSH(servers, args []string) error {
	combinedServers := strings.Join(servers, ",")

	command := append([]string{}, "-w "+combinedServers)
	command = append(command, args...)

	return r.Exec("pdsh", command)
}

func (r SSHcmd) ExecSSH(server string, args []string) error {
	command := append([]string{}, server)
	command = append(command, args...)
	return r.Exec("ssh", command)
}
