//go:build windows && host

package servers

import "github.com/pufferpanel/pufferpanel/v3/servers/standard"

func init() {
	envMapping["host"] = standard.EnvironmentFactory{}
	envMapping["standard"] = standard.EnvironmentFactory{}
}
