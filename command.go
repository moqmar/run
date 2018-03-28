package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/logrusorgru/aurora"
)

type command struct {
	description string
	usage       string
	cmd         []string
	environment map[string]string
}

func executeCommand(cmdName string) int {
	if !manualConfig {
		fmt.Print("\n")
	}

	cmd := config[cmdName]
	if len(cmd.description) > 0 {
		fmt.Printf("%s\n", Blue("[").Bold().String()+Bold(cmdName).String()+" "+Blue(cmd.description).Bold().String()+Blue("]").Bold().String())
	} else {
		fmt.Printf("%s\n", Blue("[").Bold().String()+Bold(cmdName).String()+Blue("]").Bold().String())
	}

	env := make([]string, 0)
	for _, v := range os.Environ() {
		env = append(env, v)
	}
	for n, v := range config["env"].environment {
		env = append(env, n+"="+v)
	}
	for n, v := range cmd.environment {
		env = append(env, n+"="+v)
	}
	for _, v := range getLocalEnvironment(cmdName) {
		env = append(env, v)
	}
	for _, c := range cmd.cmd {
		fmt.Printf("%s\n", Bold(Blue("» "+wrap(c, 2, 0))))
		a := append([]string{"-c", c, "--"}, args[:]...)
		child := exec.Command("/bin/sh", a...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Env = env
		child.Dir = root
		err := child.Run()
		if err != nil {
			fmt.Printf("%s\n", Bold(Red("["+err.Error()+"]")))
			if strings.HasPrefix(err.Error(), "exit status ") {
				code, err := strconv.Atoi(err.Error()[12:])
				if err == nil {
					return code
				}
			}
			return 126
		}
	}
	return 0
}

func getLocalEnvironment(cmd string) []string {
	return []string{}
}
