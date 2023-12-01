package runnerd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/oharai/self-hosted-runner-daemon/internal/github"
)

type Command struct {
}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) Execute(args []string) error {
	if len(args) < 7 {
		return fmt.Errorf("invalid arguments")
	}

	version := args[0]
	runnerOS := args[1]
	arch := args[2]
	workDir := args[3]
	repo := args[4]
	githubToken := args[5]
	labels := args[6]

	if err := c.init(version, runnerOS, arch, workDir); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	log.Printf("starting runnerd on port 8080")
	http.HandleFunc("/run", c.runHandler(workDir, repo, labels, githubToken))

	srv := &http.Server{Addr: ":8080"}

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP Server Shutdown Error: %v", err)
		}
		close(idleConnectionsClosed)
	}()

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server ListenAndServe Error: %v", err)
	}

	<-idleConnectionsClosed

	log.Printf("Bye bye")
	return nil
}

func (c *Command) init(version string, runnerOS string, arch string, workDir string) error {
	if _, err := os.Stat(workDir); err == nil {
		if err := os.RemoveAll(workDir); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
	}
	if err := os.Mkdir(workDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.Chdir(workDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	return github.DownloadRunnerTools(version, runnerOS, arch, workDir)
}

func (c *Command) runHandler(workDir, repo, labels, githubToken string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if github.IsRunnerRunning(workDir) {
			log.Printf("runner is already running")
			w.WriteHeader(http.StatusConflict)
			return
		}

		token, err := github.GetGitHubRunnerRegistrationToken(repo, githubToken)
		if err != nil {
			log.Printf("failed to get registration token: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := github.StartRunner(workDir, repo, labels, token); err != nil {
			log.Printf("failed to start runner: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
