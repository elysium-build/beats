package shell

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/libbeat/common/match"
)

type Config struct {
	Name          string              `config:"name"`
	Commands      []string            `config:"commands" validate:"required"`
	Timeout       time.Duration       `config:"timeout"`
	CustomeFields []string            `config:"custom"`
	Mode          monitors.IPSettings `config:",inline"`

	Check checkConfig `config:"check"`
}

type checkConfig struct {
	Request  requestParameters  `config:"request"`
	Response responseParameters `config:"response"`
}

type requestParameters struct {
	Method string `config:"method"` // http request method
}

type responseParameters struct {
	Ok       []match.Matcher `config:"ok"`
	Critical []match.Matcher `config:"critical"`
}

var defaultConfig = Config{
	Timeout: 16 * time.Second,
	Mode:    monitors.DefaultIPSettings,
	Check: checkConfig{
		Request: requestParameters{
			Method: "shell",
		},
		Response: responseParameters{},
	},
}

// Validate validates of the Config object is valid or not
func (c *Config) Validate() error {
	if len(c.Commands) == 0 {
		return fmt.Errorf("commands is a mandatory parameter")
	}

	// for i := 0; i < len(c.Commands); i++ {
	// 	command := c.Commands[i]
	// }

	return nil
}
