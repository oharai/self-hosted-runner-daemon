package cmd

import (
	"fmt"
	"github.com/oharai/self-hosted-runner-daemon/cmd/runnerd"
	"os"
)

type RootCmd struct {
	runnerd *runnerd.Command
}

func NewRootCmd() *RootCmd {
	return &RootCmd{}
}

func (c *RootCmd) Execute() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("no command specified")
	}

	switch os.Args[1] {
	case "runnerd":
		c.runnerd = runnerd.NewCommand()
		if err := c.runnerd.Execute(os.Args[2:]); err != nil {
			return fmt.Errorf("failed to execute runnerd: %w", err)
		}
	}

	return nil
}
