package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_docker_run_commnad(t *testing.T) {

	client := NewDockerClient()
	client.Endpoint = "tcp://20.228.150.232:2375"
	client.Timeout = 60 * time.Second
	client.Filter = []string{"name:mynginx1"}
	output, err := client.Run("/tmp", "whoami")

	assert.EqualValues(t, output, "root")
	// err := client.UploadFile("/tmp/test2/test11.txt", "/opt/test2/", "0755")
	assert.NoError(t, err, "Failed by uploading file")
}
