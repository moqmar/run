package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	. "github.com/logrusorgru/aurora"
	"gopkg.in/yaml.v2"
)

var rawConfig map[string]interface{}
var root string

var config = make(map[string]*command)

var shortExp = regexp.MustCompile(`^(\S+)\s*(.+|)$`)
var multipleExp = regexp.MustCompile(`^\S+,\S+$`)

// writeConfigPart writes a part of rawConfig to config
func writeConfigPart(cmd string, part interface{}) {
	if len(cmd) == 0 {
		fmt.Printf("%s %#v\n", Bold(Brown("Got a config section without command name:")), part)
		return
	}
	if config[cmd] == nil {
		config[cmd] = &command{}
	}

	switch c := part.(type) {
	case string:
		config[cmd].cmd = []string{c}
	case []string:
		config[cmd].cmd = c
	case map[interface{}]interface{}:
		l := 0
		if c["description"] != nil {
			l++
			switch x := c["description"].(type) {
			case string:
				config[cmd].description = x
			default:
				fmt.Printf("%s\n", Bold(Brown("description must be a string ("+cmd+")")))
			}
		}
		if c["usage"] != nil {
			l++
			switch x := c["usage"].(type) {
			case string:
				config[cmd].usage = x
			default:
				fmt.Printf("%s\n", Bold(Brown("usage must be a string ("+cmd+")")))
			}
		}
		if c["command"] != nil {
			l++
			switch x := c["command"].(type) {
			case string:
				config[cmd].cmd = []string{x}
			case []string:
				config[cmd].cmd = x
			default:
				fmt.Printf("%s\n", Bold(Brown("command must be a string or a string array ("+cmd+")")))
			}
		}
		if c["env"] != nil {
			l++
			switch x := c["env"].(type) {
			case map[interface{}]interface{}:
				if config[cmd].environment == nil {
					config[cmd].environment = make(map[string]string)
				}
				for n, v := range x {
					switch name := n.(type) {
					case string:
						switch value := v.(type) {
						case string:
							config[cmd].environment[name] = value
						case int:
							config[cmd].environment[name] = strconv.Itoa(value)
						default:
							fmt.Printf("%s\n", Bold(Brown("Environment variables must be a string ("+cmd+": "+name+")")))
						}
					default:
						fmt.Printf("%s\n", Bold(Brown("env keys must be strings ("+cmd+")")))
					}
				}
			default:
				fmt.Printf("%s\n", Bold(Brown("env must be an object ("+cmd+")")))
			}
		}
		if len(c) > l {
			fmt.Printf("A config section for %s contains unexpected keys:\n%#v", cmd, c)
		}
	}
}

// parseConfig parses the rawConfig to config, a map[string]*command
func parseConfig() {
	for key, content := range rawConfig {
		if key == "env" {
			writeConfigPart("env", map[interface{}]interface{}{"env": content})
		} else if multipleExp.MatchString(key) {
			appliesTo := strings.Split(key, ",")
			for _, cmd := range appliesTo {
				if len(cmd) > 0 {
					writeConfigPart(cmd, content)
				}
			}
		} else {
			parts := shortExp.FindStringSubmatch(key)
			if len(strings.TrimSpace(parts[2])) > 0 {
				writeConfigPart(parts[1], map[interface{}]interface{}{"description": strings.TrimSpace(parts[2])})
			}
			writeConfigPart(parts[1], content)
		}
	}
	if config["env"] == nil {
		config["env"] = &command{}
	}
}

// getConfig parses the configFile to rawConfig
func getConfig(configFile string) {
	if configFile == "" { // Get the config file by walking up the file tree
		cwd, err := os.Getwd()
		if err == nil {
			p := "."
			for n := strings.Count(strings.TrimSuffix(cwd, "/"), "/"); n > 0; n-- {
				if _, err := os.Stat(p + "/.run"); err == nil {
					f, err := filepath.Abs(p + "/.run")
					if err == nil {
						configFile = f
						break
					}
				}
				p += "/.."
			}
		}
	} else {
		if stat, err := os.Stat(configFile); os.IsNotExist(err) || stat.IsDir() {
			fmt.Printf("%s\n", Red(Bold("The specified config file doesn't exist.")))
			os.Exit(1)
		}
	}

	if configFile == "" {
		help()
		os.Exit(1)
	}

	dat, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("%s\n", Red(Bold(err.Error())))
		os.Exit(1)
	}
	err = yaml.Unmarshal(dat, &rawConfig)
	if err != nil {
		fmt.Printf("%s\n", Red(Bold(err.Error())))
		os.Exit(1)
	}

	root = filepath.Dir(configFile)
	os.Chdir(filepath.Dir(configFile))
}
