package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/logrusorgru/aurora"
)

var args []string

func main() {
	args = os.Args[1:]

	configFile := ""
	if len(args) >= 2 && args[0] == "-c" {
		configFile = args[1]
		args = args[2:]
	}
	getConfig(configFile)
	parseConfig()

	if len(args) == 0 {
		run()
	}

	switch args[0] {
	case "--help":
		fallthrough
	case "-h":
		fallthrough
	case "help":
		help()
	case "env":
		args = args[1:]
		env()
	// TODO: -a, -d
	default:
		run()
	}
}

func help() {
	if len(config) > 0 {
		fmt.Printf("%s\n", Green("Available commands:"))
		cmdlen := 0
		for cmd := range config {
			if cmd != "env" && len(cmd) > cmdlen {
				cmdlen = len([]rune(cmd))
			}
		}
		for cmd, info := range config {
			if cmd != "env" {
				fmt.Printf(" %"+strconv.Itoa(cmdlen)+"s  %s\n", Bold(Blue(cmd)), Bold(wrap(info.description, cmdlen+3, 80)))
				// TODO: Usage
				// TODO: Commands (-c)
			}
		}
	}
	if len(config) == 0 || (len(args) > 0 && (args[0] == "--help" || args[0] == "-h")) || (len(args) > 1 && args[1] == "-v") {
		fmt.Printf("%s\n", Green("The run command:"))
		fmt.Printf("%s", Blue(` help               `).String()+`print available commands`+"\n"+
			Blue(` help -v | --help   `).String()+`additionally print help for the run command itself`+"\n"+
			Blue(` help <cmd>         `).String()+`completely print a command and its help`+"\n"+
			Blue(` help -c            `).String()+`completely print all available commands`+"\n"+
			//Blue(` env                `).String()+`print all environment variables`+"\n"+
			//Blue(` env edit           `).String()+`edit local environment variables`+"\n"+
			//Blue(` env reset          `).String()+`reset all local environment variables`+"\n"+
			//Blue(` -a <cmd> [script]  `).String()+`add a new command`+"\n"+
			//Blue(` -d <cmd>           `).String()+`remove a command`+"\n"+
			Blue(` -c <.run> [...]    `).String()+`use a custom .run file`+"\n")
		fmt.Printf("%s\n", Bold(Green("run v0.3 - https://github.com/moqmar/run")))

	}
}

func env() {
	// TODO: env edit
	// TODO: env reset
	k := 0
	if config["env"] != nil && config["env"].environment != nil {
		for n, v := range config["env"].environment {
			fmt.Printf("%s='%s'\n", n, strings.Replace(v, `'`, `'"'"'`, -1))
			k++
		}
	}
}

func run() {
	if len(args) > 0 && config[args[0]] != nil {
		cmdName := args[0]
		args = args[1:]
		executeCommand(cmdName)
	} else if config["run"] != nil {
		executeCommand("run")
	} else {
		help()
	}
	os.Exit(0)
}
