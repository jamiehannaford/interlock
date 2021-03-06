package interlock

import (
	"github.com/jamiehannaford/interlock/Godeps/_workspace/src/github.com/samalba/dockerclient"
)

type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Url         string
}

type Plugin interface {
	Info() *PluginInfo
	Init() error
	HandleEvent(event *dockerclient.Event) error
}
