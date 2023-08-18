package shell

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func FakeCommand(command string, args ...string) ExecutionRequest {
	cs := []string{command}
	cs = append(cs, args...)
	cmdStr := strings.Join(cs, " ")
	trimmedCmd := strings.Trim(cmdStr, " ")
	env := []string{"GO_WANT_HELPER_PROCESS=1"}

	execution := ExecutionRequest{
		Command: trimmedCmd,
		Env:     env,
	}

	return execution
}

func CleanOutput(s string) string {
	return strings.Replace(s, "\r", "", -1)
}

func TestExecute(t *testing.T) {
	echo := FakeCommand("echo", "foo")

	echoExec, echoErr := echo.Execute(context.Background(), echo)
	assert.Equal(t, nil, echoErr)
	assert.Equal(t, "foo\n", echoExec.Output)
	assert.Equal(t, 0, echoExec.Status)
	assert.NotEqual(t, 0, echoExec.Duration)

	sleep := FakeCommand("sleep 10")
	sleep.Timeout = 5 * time.Second

	sleepExec, sleepErr := sleep.Execute(context.Background(), sleep)
	assert.Equal(t, nil, sleepErr)
	assert.Equal(t, "Execution timed out\n", CleanOutput(sleepExec.Output))
	assert.Equal(t, 2, sleepExec.Status)
	assert.NotEqual(t, 0, sleepExec.Duration)
}
