package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var Version string

func main() {
	if len(os.Args) <= 1 {
		pinfo(fmt.Sprintf("This is EKSphemeral in version %v", Version))
		perr("Please specify one of the following commands: install, uninstall, create, list, or prolong", nil)
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "install", "i":
		pinfo("Trying to install EKSphemeral ...")
		shellout("./eksp-up.sh")
	case "uninstall", "u":
		pinfo("Trying to uninstall EKSphemeral ...")
		shellout("./eksp-down.sh")
	case "create", "c":
		pinfo("Trying to create a new ephemeral cluster ...")
		if len(os.Args) > 2 {
			pinfo("... using cluster spec " + os.Args[2])
			if _, err := os.Stat(os.Args[2]); os.IsNotExist(err) {
				perr("Can't create a cluster due to invalid spec:", err)
				os.Exit(2)
			}
			shellout("./eksp-create.sh", os.Args[2])
			break
		}
		// creating cluster with defaults:
		shellout("./eksp-create.sh", os.Args[2])
	case "list", "ls", "l":
		if len(os.Args) > 2 { // we have a cluster ID, try looking up cluster spec
			clusterSpecFile := os.Args[2]
			shellout("./eksp-list.sh", clusterSpecFile)
			break
		}
		// listing all cluster:
		shellout("./eksp-list.sh")
	case "prolong", "p":
		if len(os.Args) < 4 {
			perr("Can't prolong cluster lifetime without both the cluster ID and the time in minutes provided", nil)
			os.Exit(3)
		}
		cID := os.Args[2]
		prolongFor := os.Args[3]
		shellout("./eksp-prolong.sh", cID, prolongFor)
	default:
		perr("Please specify one of the following commands: install, uninstall, create, list, or prolong", nil)
	}
}

// shellout shells out to execute a command with a variable number
// of arguments and prints the literal results from both stdout and stderr
func shellout(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		perr("Can't shell out due to issues with stderr:", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		perr("Can't shell out due to issues with stdout:", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		perr("Can't shell out due to issues with starting command:", err)
		return
	}
	go echo(stderr)
	go echo(stdout)
	err = cmd.Wait()
	if err != nil {
		perr("Something bad happened after command completed:", err)
	}
}

// echo prints the character stream as a set of lines
func echo(rc io.ReadCloser) {
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

// pinfo writes msg in light blue to stderr
// see also https://misc.flogisoft.com/bash/tip_colors_and_formatting
func pinfo(msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "\x1b[94m%v\x1b[0m\n", msg)
}

// perr writes message (and optionally error) in light red to stderr
// see also https://misc.flogisoft.com/bash/tip_colors_and_formatting
func perr(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: \x1b[91m%v\x1b[0m\n", msg, err)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\x1b[91m%v\x1b[0m\n", msg)
}
