package dead

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/ErikDubbelboer/gspt"
	"github.com/fsnotify/fsnotify"
)

type Config struct {
	Env      string
	Patterns []string
	Debounce time.Duration
}

func Default() *Config {
	return &Config{
		Env:      "DEAD",
		Patterns: make([]string, 0),
		Debounce: 500 * time.Millisecond,
	}
}

func (c *Config) Watch(patterns ...string) *Config {
	c.Patterns = append(c.Patterns, patterns...)
	return c
}

func (c *Config) Main() {
	if c.Env != "" && os.Getenv(c.Env) == "watch" {
		// Change environment variable so we don't end up in a loop
		os.Setenv(c.Env, "main")

		// Make it more clear in process list which is the watcher process
		gspt.SetProcTitle(os.Args[0] + " (watch)")

		// TODO: make this configurable
		pipeline := [][]string{
			{".go", "go", "build"},
			{".html", "", ""},
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
						ext := filepath.Ext(event.Name)
						for i := range pipeline {
							if pipeline[i][0] == ext {
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
					/* FIXME:
					if !debounce.Stop() {
						<-debounce.C
					}
					*/
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
						bin := pipeline[i][1]
						args := pipeline[i][2:]
						if pipeline[i][1] != "" {
							log.Printf("Building!")
							cmd2 := exec.Command(bin, args...)
							cmd2.Stdout = os.Stdout
							cmd2.Stderr = os.Stderr
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

		// Shouldn't reach this
		os.Exit(1)
	}
}

func startCommand() *exec.Cmd {
	log.Printf("Starting!")

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	return cmd
}

func stopCommand(cmd *exec.Cmd) {
	if cmd != nil {
		log.Printf("Stopping!")

		if err := cmd.Process.Kill(); err != nil {
			panic(err)
		}

		cmd.Wait()
	}
}
