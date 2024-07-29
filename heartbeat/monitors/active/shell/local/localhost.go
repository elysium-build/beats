package local

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type LocalClient struct {
	Timeout time.Duration
}

func NewLocalClient() *LocalClient {
	return &LocalClient{}
}

func (c *LocalClient) Run(dir, command string, args ...string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()

	switch err.(type) {
	case *exec.ExitError:
		if strings.HasPrefix(err.Error(), "exit status") {
			return strings.Trim(fmt.Sprintf("%v. %v ", strings.Trim(string(output), "\n"), err.Error()), "\n"), nil
		}
	}
	return string(output), err
}

func (c *LocalClient) Connect() error {
	return nil
}
func (c *LocalClient) Reconnect() error {
	return nil
}
func (c *LocalClient) Close() {}
