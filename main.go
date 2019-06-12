package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var Version string

func main() {
	if len(os.Args) <= 1 {
		pinfo(fmt.Sprintf("This is EKSphemeral in version %v", Version))
		perr("Please specify one of the following commands: install, uninstall, create, list, or prolong", nil)
		os.Exit(-1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "install", "i":
		pinfo("Trying to install EKSphemeral ...")
		r, _ := shellout(true, false, "./eksp-up.sh", "-al")
		fmt.Println(r)
	case "uninstall", "u":
		pinfo("Trying to uninstall EKSphemeral ...")
		r, _ := shellout(true, false, "./eksp-down.sh", "-al")
		fmt.Println(r)
	case "create", "c":
		pinfo("Trying to create a new ephemeral cluster ...")
		if len(os.Args) > 2 {
			pinfo("... using cluster spec " + os.Args[2])
			if _, err := os.Stat(os.Args[2]); os.IsNotExist(err) {
				perr("Can't create a cluster due to invalid spec:", err)
				os.Exit(-1)
			}
			r, _ := shellout(true, false, "./eksp-create.sh", os.Args[2])
			fmt.Println(r)
			break
		}
		// creating cluster with defaults:
		r, _ := shellout(true, false, "./eksp-create.sh", os.Args[2])
		fmt.Println(r)
	case "list", "ls", "l":
		if len(os.Args) > 2 { // we have a cluster ID, try looking up cluster spec
			clusterSpecFile := os.Args[2]
			r, _ := shellout(true, false, "./eksp-list.sh", clusterSpecFile)
			fmt.Println(r)
			break
		}
		// listing all cluster:
		r, _ := shellout(true, false, "./eksp-list.sh")
		fmt.Println(r)
	case "prolong", "p":
		if len(os.Args) < 4 {
			perr("Can't prolong cluster lifetime without both the cluster ID and the time in minutes provided", nil)
			os.Exit(-1)
		}
		cID := os.Args[2]
		prolongFor := os.Args[3]
		r, _ := shellout(true, false, "./eksp-prolong.sh", cID, prolongFor)
		fmt.Println(r)
	default:
		perr("Please specify one of the following commands: install, uninstall, create, list, or prolong", nil)
	}
}

// shellout shells out to execute a command with a variable number
// of arguments and returns the literal result. Optionally, you can
// including stderr output and echoing the command when verbose is true.
func shellout(withstderr, verbose bool, cmd string, args ...string) (result string, err error) {
	var out bytes.Buffer
	if verbose {
		pinfo(cmd + " " + strings.Join(args, " "))
	}
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	if withstderr {
		c.Stderr = os.Stderr
	}
	c.Stdout = &out
	err = c.Run()
	if err != nil {
		return "", err
	}
	result = strings.TrimSpace(out.String())
	return result, nil
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
