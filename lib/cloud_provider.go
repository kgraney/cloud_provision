package cloud_provision

import (
	"github.com/codegangsta/cli"
)

type CloudProvider interface {
	GetCommands() []cli.Command
}
