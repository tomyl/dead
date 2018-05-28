// Package dead makes it easy for web servers to restart on source code or template changes.
package dead

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/ErikDubbelboer/gspt"
	"github.com/fsnotify/fsnotify"
)

// Config says what directories to watch and what to execute when building.
type Config struct {
	// Which environment variable to inspect when checking if we should start watching.
	Env string
	// What directories to watch. Can use glob patterns.
	Patterns []string
	// How long to wait before acting on file modification event.
	Debounce time.Duration
	// What command to execute when building.
	BuildPath string
	BuildArgs []string
}

// Default returns a reasonable default config (environment variable DEAD, 500
// ms debounce time, run "go build" to build.
func Default() *Config {
	return &Config{
		Env:       "DEAD",
		Patterns:  make([]string, 0),
		Debounce:  500 * time.Millisecond,
		BuildPath: "go",
		BuildArgs: []string{"build"},
	}
}

// Watch adds directories to watch for file changes. Can use glob patterns.
func (c *Config) Watch(patterns ...string) *Config {
	c.Patterns = append(c.Patterns, patterns...)
	return c
}

// Main is the watch main loop. Will return immediately if environment variable
// isn't set to "watch".
func (c *Config) Main() {
	if c.Env != "" && os.Getenv(c.Env) == "watch" {
		// Change environment variable so we don't end up in a loop
		os.Setenv(c.Env, "")

		// Make it more clear in process list which is the watcher process
		gspt.SetProcTitle(os.Args[0] + " (watch)")

		type step struct {
			suffix string
			path   string
			args   []string
		}

		// TODO: make this configurable
		pipeline := []step{
			{".go", c.BuildPath, c.BuildArgs},
			{".html", "", nil},
		}

		// Starting watching filesystem
		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			panic(err)
		}

		defer watcher.Close()

		actions := make(chan int)

		go func() {
			for {
				select {
				case event := <-watcher.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {
						for i := range pipeline {
							if strings.HasSuffix(event.Name, pipeline[i].suffix) {
								actions <- i
								break
							}
						}
					}
				case err := <-watcher.Errors:
					if err != nil {
						panic(err)
					}
				}
			}
		}()

		for _, pattern := range c.Patterns {
			names, err := filepath.Glob(pattern)
			if err != nil {
				panic(err)
			}
			for _, name := range names {
				if err = watcher.Add(name); err != nil {
					panic(err)
				}
			}
		}

		// Start myself
		cmd := startCommand()

		defer func() {
			if cmd != nil {
				stopCommand(cmd)
			}
		}()

		// Listen for Ctrl-C
		intr := make(chan os.Signal, 1)
		signal.Notify(intr, os.Interrupt)

		debounce := time.NewTimer(c.Debounce)
		nextAction := -1

		for {
			select {
			case <-intr:
				stopCommand(cmd)
				os.Exit(1)

			case action := <-actions:
				if nextAction < 0 || action < nextAction {
					debounce.Reset(c.Debounce)
					nextAction = action
				}

			case <-debounce.C:
				if nextAction >= 0 {
					// Source code was updated. Stop command and rebuild if necessary
					// and then start command again.
					stopCommand(cmd)
					cmd = nil
					for i := nextAction; i < len(pipeline); i++ {
						path := pipeline[i].path
						args := pipeline[i].args
						if path != "" && args != nil {
							cmd2 := exec.Command(path, args...)
							cmd2.Stdout = os.Stdout
							cmd2.Stderr = os.Stderr
							log.Printf("Building %s", cmd2.Args)
							if err := cmd2.Run(); err != nil {
								log.Printf("Build failed!")
								break
							}
						} else {
							cmd = startCommand()
							break
						}
					}
					nextAction = -1
				}
			}
		}

		// Unreachable
	}
}

func startCommand() *exec.Cmd {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("Starting %s", cmd.Args)

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	return cmd
}

func stopCommand(cmd *exec.Cmd) {
	if cmd != nil {
		log.Printf("Stopping")

		if err := cmd.Process.Kill(); err != nil {
			panic(err)
		}

		cmd.Wait()
	}
}
