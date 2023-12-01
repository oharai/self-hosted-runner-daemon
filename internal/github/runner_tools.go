package github

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/oharai/self-hosted-runner-daemon/util"
)

func DownloadRunnerTools(version, runnerOS, arch, workDir string) error {
	// Download runner package
	fileName := fmt.Sprintf("actions-runner-%s-%s-%s.tar.gz", runnerOS, arch, version)
	url := fmt.Sprintf("https://github.com/actions/runner/releases/download/v%s/%s", version, fileName)
	pathToDownload := path.Join(workDir, fileName)
	if err := util.DownloadFile(pathToDownload, url); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Extract archive
	cmd := exec.Command("tar", "xzf", pathToDownload)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	return nil
}

func StartRunner(workDir, repo, labels, registrationToken string) error {
	cmd := exec.Command(
		workDir+"/config.sh",
		"--url", "https://github.com/"+repo,
		"--token", registrationToken,
		"--labels", labels,
		"--disableupdate",
		"--replace",
		"--unattended",
		"--ephemeral",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup runner: %w", err)
	}

	// Start runner
	cmd = exec.Command(workDir + "/run.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run runner: %w", err)
	}

	return nil
}

func IsRunnerRunning(workDir string) bool {
	// workDir/.runner exists if runner is running
	_, err := os.Stat(workDir + "/.runner")
	return err == nil
}
