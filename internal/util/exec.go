package util

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand executes a shell command with the given arguments
// and returns its stdout as a []byte.
// If an error occurs the content of stderr is printed
// and an error is returned.
func ExecCommand(command string, args ...string) (string, error) {
	//log.Printf("Executing command: %s %s", command, args)
	cmd := exec.Command(command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(stderr.String())
		return "", err
	}

	result := stdout.String()
	result = strings.TrimSpace(result)

	return result, nil
}

// ExecCommandEnv is like ExecCommand but with the possibility to add environment variables
// to the executed process.
func ExecCommandEnv(env []string, attach bool, command string, args ...string) (string, error) {
	//log.Printf("Executing command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	var err error
	if attach {
		err = cmd.Run()
	} else {
		err = cmd.Start()
		if err != nil {
			fmt.Println(err.Error())
			return "", err
		}
		err = cmd.Process.Release()
	}

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(stderr.String())
		log.Fatal(stderr)
		return "", err
	}

	return stdout.String(), nil
}
