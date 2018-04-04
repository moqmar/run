package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	. "github.com/logrusorgru/aurora"
)

type command struct {
	description string
	usage       string
	replace     string
	before      string
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
	if strings.HasPrefix(cmd.replace, "ssh ") {
		child := exec.Command("ssh-agent", "-s")
		child.Env = env
		child.Dir = root
		if out, err := child.Output(); err == nil {
			env = append(env, regexp.MustCompile(`SSH_(AUTH_SOCK=[^\s;]+|AGENT_PID=\d+)`).FindAllString(string(out), -1)...)
		}
		child = exec.Command("ssh-agent", "-k")
		child.Env = env
		child.Dir = root
		defer child.Run()
	}
	if cmd.before != "" {
		child := exec.Command("/bin/sh", append([]string{"-c", cmd.before, "--"}, args[:]...)...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Env = env
		child.Dir = root
		err := child.Run()
		if err != nil {
			fmt.Printf("Error during `before` command: %s\n", Bold(Red(err.Error())))
			fmt.Print("\n")
			return 126
		}
	}
	status := 0
	for _, c := range cmd.cmd {
		fmt.Printf("%s\n", Bold(Blue("» "+wrap(c, 2, 0))))
		var a []string
		if cmd.replace != "" {
			a = append([]string{"-e", "-c", cmd.replace, "--"}, args[:]...)
		} else {
			a = append([]string{"-e", "-c", c, "--"}, args[:]...)
		}
		child := exec.Command("/bin/sh", a...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Env = env
		child.Dir = root
		if cmd.replace != "" {
			child.Stdin = strings.NewReader(c)
		}
		err := child.Run()
		if err != nil {
			fmt.Printf("%s\n", Bold(Red("["+err.Error()+"]")))
			if strings.HasPrefix(err.Error(), "exit status ") {
				code, err := strconv.Atoi(err.Error()[12:])
				if err == nil {
					status = code
					break
				}
			}
			status = 126
			break
		}
	}
	fmt.Print("\n")
	return status
}

func getLocalEnvironment(cmd string) []string {
	return []string{}
}
