package runnerd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"

	"github.com/oharai/self-hosted-runner-daemon/internal/github"
	"github.com/oharai/self-hosted-runner-daemon/util"
)

type Command struct {
}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) Execute(args []string) error {
	if len(args) < 6 {
		return fmt.Errorf("invalid arguments")
	}

	version := args[0]
	runnerOS := args[1]
	arch := args[2]
	workDir := args[3]
	repo := args[4]
	githubToken := args[5]

	if err := c.init(version, runnerOS, arch, workDir); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	log.Printf("starting runnerd on port 8080")
	http.HandleFunc("/setup", c.setupHandler(workDir, repo, githubToken))

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

func (c *Command) init(
	version string,
	runnerOS string,
	arch string,
	workDir string,
) error {
	// ディレクトリを作成し、その中に移動する
	err := os.Mkdir(workDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	err = os.Chdir(workDir)
	if err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// ランナーパッケージをダウンロードする
	fileName := fmt.Sprintf("actions-runner-%s-%s-%s.tar.gz", runnerOS, arch, version)
	url := fmt.Sprintf("https://github.com/actions/runner/releases/download/v%s/%s", version, fileName)
	log.Printf("downloading %s", url)
	err = util.DownloadFile(fileName, url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// アーカイブを解凍する
	cmd := exec.Command("tar", "xzf", fileName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	return nil
}

func (c *Command) setupHandler(workDir, repo, githubToken string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := github.GetGitHubRunnerRegistrationToken(repo, githubToken)
		if err != nil {
			log.Printf("failed to get registration token: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// ランナーの設定を開始する
		cmd := exec.Command(workDir+"/config.sh", "--ephemeral", "--url", "https://github.com/"+repo, "--token", token)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("failed to setup runner: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// ランナーを実行する
		cmd = exec.Command(workDir + "/run.sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("failed to run runner: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
