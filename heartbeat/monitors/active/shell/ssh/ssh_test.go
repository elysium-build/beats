package ssh

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	keydata = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1eZfvFCwbKH6KM9/ivgNOIqFCL3JqL+CLXSu6On9POQzPTXo
whKJ5D7Xl4DbNxJSVHmf6J2bhRhosX5Qud8eJuvFcI2Ep7E2sPXLMSHPYkvz9fGj
0SETyOoE1thqMCabez+hj058nU3L/x6rd5MM2XWTeyqLvYSAR357gYRj8i+8OVYV
nrSAdRW43jyt0I+BI7H9z6WDgTqvWa60dV4D6PqhYKYoPsfYxHlPUfIxcNcgknjx
7M+pf9Gv6R/NZ8nViihskFs2OFbdiFcNHz1eKtxSNXz1F20bfhwIXYOJB0RsB/rE
TSCztaMtoXfiPCFCn3qV1uXH3ZjgK8/X3emeAwIDAQABAoIBAAhB5nw8mTL4ZdHh
gMj6nngKUOxvdzN+gSYEFSSEs/P/00KPrDahxJT9IBGHNe9AU9FTCKtQOkq/EHuZ
psAmLuHNxEd+DxryKmxWcMuqxHjE+dwKwgo2vq7I6frpS+Aj/WiaokAIBaOE91UX
+AKbuKlEcrcUg2SDkvgvl9D+LWWSBB171Sk7GCB4TPXJRkWjg0ogtBEel4MccDgS
TptgWMwA70uX4EU0LERKCq62/Bh1yauTxnmy/LXTt/McVjNv+F1fWoMxVRQ/WOvL
VvS0RZh7ak4WKANQUyPFyzaJm8rjtPArM9pnt01HMF7XKWSQo1S4Wij4njZSXcQ4
2mkQBeECgYEA7+NgmjHFCAyk8gqSeQs3Tr9d9G7wpHyKJJD8f6oBywMgwAnVpNNl
MLTTksh75KPyPyY4pDtvq512lgdXZ3BL69GbqInxMhAzE3EYnLMXzebT6wbqk4bQ
7QgrhdS+oQ01Qa5Cgv226bWAm8ve3wJHR8dIi8k0LT3OdoaZKZxmJrsCgYEA5EQn
zrHrFUo+cZ6eqkGfVnNN8h956skyiE3NqPzYQivmF+0O24tQsvFCra/yRZI31va6
u6JAYV5dwSjJRwba0k/zyNOic00hcfGTj8kdCJm8bx/0UaGTOzNDyYh70Q70RHgP
rWqTGj3PBpmKaxH7Leg3Am6NDrC7OdN+dWXlhVkCgYEA1Wmzx3n/j+mv1KUTKhyQ
V75oF82ayLsDKwTRncHhVnqx6CbXqotmuq4ki7FQh1hTa1rViUZXUpYDqfVeDOga
ovEXShluOtuulN1IyB+MTeHNJopApn6J4FYkYiuibCUT/BrLkT2mPMT8ZZ456Kxe
Pb1NDQ8zHAygYVHdcOdy+YECgYEAqwT5Qh4AwCGw6RVrUKn7xBx9YJL+l86IAqEw
HZTaPbGAIZrlT81v97FUQKca/87N8UtHmj60t36pBYgWTRWwqnNmdadCBdra3PCe
mtKV4xSznho1xVcl5OvCtOKByZ7HmejN7iJz9ewrCInOr+t34ewiErtbCY+Vpnxz
OWfPb3kCgYBc5q+JaZX2uFdBr566dsN3iu2HfgUm5rqCzgHYeqd9YpHVVqMp36V8
A60lHNWh6HZRO0kXqu43lYPAe8rxyLLDgGi29uU+Tar2dqUpBsisVJvQUKBVQRIC
oQ/3fM04gVOMWvSYh4OqPZzOhfdZn+PqgsG/9vmJBSJV3FnH9RfjcA==
-----END RSA PRIVATE KEY-----`
)

func Test_run_output(t *testing.T) {
	comm := &SSHClient{
		Addr:       "20.228.150.232:22",
		Username:   "test",
		Password:   "H4rrykarmus!",
		Timeout:    2 * time.Second,
		initClient: &sync.Once{},
	}

	defer comm.Run("~", "rm test.sh")
	comm.Run("~", "echo '#!/bin/sh' >test.sh")
	comm.Run("~", "echo 'echo test test1' >test.sh")
	comm.Run("~", "chmod 755 test.sh")
	out, err := comm.Run("~", "/bin/bash", "test.sh")

	assert.Equal(t, "test test1", out)
	assert.NoError(t, err, "Failed by running comm.Output")
}

func Test_timeout(t *testing.T) {
	comm := &SSHClient{
		Addr:       "20.228.150.232:22",
		Username:   "test",
		Password:   "H4rrykarmus!",
		Timeout:    2 * time.Second,
		initClient: &sync.Once{},
	}

	_, err := comm.Run("", "sleep 5")

	assert.EqualError(t, err, "Connection is disconnected by the timeout or lost")
	out, err := comm.Run("", "echo", "test", "test1")
	assert.Equal(t, "test test1", out)
	assert.NoError(t, err, "Failed by running timeout test")
}

func Test_Key_file(t *testing.T) {
	comm := &SSHClient{
		Addr:       "20.228.150.232:22",
		Username:   "test",
		Password:   "H4rrykarmus!",
		Timeout:    2 * time.Second,
		initClient: &sync.Once{},
	}

	out, err := comm.Run("", "ps aux | grep ipa | wc -l")
	assert.Equal(t, "2", out)
	assert.NoError(t, err, "Failed by running timeout test")
}
