package shell

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

type Config struct {
	Name string `config:"name"`

	// connection settings
	Hosts []string `config:"hosts" validate:"required"`

	Mode monitors.IPSettings `config:",inline"`
	// authentication
	Username   string `config:"username"`
	Password   string `config:"password"`
	Sudo       bool   `config:"sudo"`
	Key        string `config:"key"`
	Occurrence int    `config:"occurrence"`
	// configure tls
	TLS *tlscommon.Config `config:"ssl"`
	// configure validation
	Check checkConfig `config:"check"`
	// CustomeFields []string      `config:"custom"`
	Timeout time.Duration `config:"timeout"`

	Docker       bool     `config:"docker"`
	Dockerfilter []string `config:"dockerfilter"`

	Upload []string `config:"upload"`

	LiveConnection bool `config:"liveconnection"`
}

type checkConfig struct {
	Request  commandConfig `config:"request"`
	Response outputConfig  `config:"output"`
}

type commandConfig struct {
	Command string   `config:"command"`
	Args    []string `config:"args"`
	Dir     string   `config:"dir"`
}

type outputConfig struct {
	Ok       []match.Matcher `config:"ok"`
	Critical []match.Matcher `config:"critical"`
}

// defaultConfig creates a new copy of the monitors default configuration.
func defaultConfig() Config {
	return Config{
		Name: "echo",
		// Hosts:          []string{"localhost:22"},
		Mode:           monitors.DefaultIPSettings,
		TLS:            nil,
		Timeout:        16 * time.Second,
		Docker:         false,
		LiveConnection: false,
		Occurrence:     3,
		Dockerfilter:   []string{},
		Check: checkConfig{
			Request: commandConfig{
				Dir: "",
			},
			Response: outputConfig{},
		},
	}
}

func (c *Config) uploadchecker() error {
	for _, uploadFile := range c.Upload {
		fromandto := strings.Split(uploadFile, ":")
		if len(fromandto) < 2 {
			return fmt.Errorf("The upload configure should be in format <SourcePath>:<DestPath> for %v", c.Name)
		}
	}
	return nil
}

func (c *Config) dockerchecker() error {
	if len(c.Dockerfilter) == 0 {
		return fmt.Errorf("The dockerfiler is required for %v ", c.Name)
	}
	if c.Sudo {
		return fmt.Errorf("docker doesn't support the sudo for %v", c.Name)
	}
	if len(c.Upload) > 0 {
		return c.uploadchecker()
	}
	return nil
}

func (c *Config) sshchecker() error {
	if c.Username == "" {
		return fmt.Errorf("Username is required for %v", c.Name)
	}

	if c.Password == "" && c.Key == "" {
		return fmt.Errorf("Either Password and key is required for %v", c.Name)
	}
	if strings.Index(c.Key, "@") == 0 {
		_, err := os.Stat(string(c.Key[1:]))
		if err != nil {
			return err
		}
	}
	if c.Sudo && c.Password == "" {
		return fmt.Errorf("Password is required for sudo for %v", c.Name)
	}

	if len(c.Upload) > 0 {
		return c.uploadchecker()
	}
	return nil
}

func (c *Config) localchecker() error {
	if c.Sudo {
		return fmt.Errorf("Localhost doesn't support the sudo for %v", c.Name)
	}
	if len(c.Upload) > 0 {
		return fmt.Errorf("Localhost doesn't support the upload for %v", c.Name)
	}
	return nil
}

func (c *Config) Validate() error {
	if c.Occurrence <= 0 {
		return fmt.Errorf("Occurrence can't be less than 0 for %v", c.Name)
	}
	if c.Docker {
		return c.dockerchecker()
	}
	for _, addr := range c.Hosts {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}

		if strings.ToLower(host) == "localhost" {
			return c.localchecker()
		}
		return c.sshchecker()

	}
	return nil
}

func (c *checkConfig) Validate() error {
	return nil
}

func (c *commandConfig) Validate() error {
	return nil
}

func (c *outputConfig) Validate() error {
	return nil

}
