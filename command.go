package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	. "github.com/logrusorgru/aurora"
	"github.com/monochromegane/go-gitignore"
)

type command struct {
	description  string
	usage        string
	replace      string
	before       string
	cmd          []string
	environment  map[string]string
	simultaneous bool
	watch        string
	watchIgnore  string
}

var running map[*exec.Cmd]bool
var runningCount = 0
var runningLock = sync.Mutex{}
var mayQuit = make(chan bool, 1) // event channel: an application has exited
var lastStatus = 0
var intentionalExit = false
var mayContinue chan bool // event channel: wait for a file update when watching, even if the processes have exited

func runCommand(cmd *command, env []string, osExit bool, killEverything bool) {
	running = make(map[*exec.Cmd]bool, len(cmd.cmd))
	mayQuit = make(chan bool, len(cmd.cmd))
	lastStatus = 0
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
		runningLock.Lock()
		running[child] = true
		runningLock.Unlock()
		if cmd.simultaneous {
			runningCount++
			go runChild(child, true)
		} else {
			if code := runChild(child, false); code != 0 {
				if osExit {
					os.Exit(code)
				} else {
					break
				}
			}
		}

		if intentionalExit {
			break
		}
	}
	if cmd.simultaneous {
		killed := false
		for {
			err := <-mayQuit
			runningCount--
			if runningCount <= 0 {
				return
			} else if err && killEverything && !killed {
				killall()
				killed = true
			}
		}
	}
}

func runChild(child *exec.Cmd, simultaneous bool) int {
	// all child processes stored in running will be killed when a watched file changes
	err := child.Run()
	runningLock.Lock()
	delete(running, child)
	runningLock.Unlock()
	if simultaneous {
		mayQuit <- err != nil
	}

	if err != nil {
		fmt.Printf("%s\n", Bold(Red("["+err.Error()+"]")))
		if strings.HasPrefix(err.Error(), "exit status ") {
			code, err := strconv.Atoi(err.Error()[12:])
			if err == nil {
				lastStatus = code
				return code
			}
		}
		lastStatus = 126
		return 126
	}
	return 0
}

func killall() {
	runningLock.Lock()
	for child := range running {
		if child.Process != nil {
			child.Process.Signal(syscall.SIGTERM)
		}
	}
	runningLock.Unlock()
}

var updating = false
var updatingLock = sync.Mutex{}

func update() {
	updatingLock.Lock()
	if updating {
		updatingLock.Unlock()
		return
	}
	updating = true
	updatingLock.Unlock()

	time.Sleep(100 * time.Millisecond)
	killall()
	mayContinue <- true

	updatingLock.Lock()
	updating = false
	updatingLock.Unlock()
}

func signalProxy() {
	// Listen for signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	intentionalExit = true
	killall()
}

func watch(cmd *command, env []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Listen for watcher events
	go func() {
		for {
			select {
			case <-watcher.Events:
				go update()
			case err := <-watcher.Errors:
				log.Println("Watcher Error:", err)
			}
		}
	}()

	// Initialize watcher
	include := gitignore.NewGitIgnoreFromReader("/", strings.NewReader(cmd.watch))
	exclude := gitignore.NewGitIgnoreFromReader("/", strings.NewReader(cmd.watchIgnore))
	ignoredDirectories := []string{}
	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		for _, d := range ignoredDirectories {
			if strings.HasPrefix(path, d) {
				return nil
			}
		}
		if f.IsDir() && exclude.Match("/"+path, true) {
			ignoredDirectories = append(ignoredDirectories, path+"/")
		} else if !f.IsDir() && include.Match("/"+path, false) {
			if err = watcher.Add(path); err != nil {
				log.Fatal(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Run stuff forever (until intentionalExit)
	mayContinue = make(chan bool, 1)
	for {
		runCommand(cmd, env, false, false)
		if intentionalExit {
			break
		} else {
			<-mayContinue
		}
	}
	os.Exit(lastStatus)
}

func executeCommand(cmdName string) {
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
			os.Exit(126)
		}
	}

	go signalProxy()
	if cmd.watch != "" {
		watch(cmd, env)
	} else {
		runCommand(cmd, env, true, true)
	}
}

func getLocalEnvironment(cmd string) []string {
	return []string{}
}
