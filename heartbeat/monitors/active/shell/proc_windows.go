// +build windows

package shell

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
)

// Command returns a command to execute a script through a shell.
func Command(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: strings.Join([]string{"/c", command}, " "),
	}
	return cmd
}

// SetProcessGroup sets the process group of the command process
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}

// KillProcess kills the command process and any child processes
func KillProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
