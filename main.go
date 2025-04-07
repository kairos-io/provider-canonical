package main

import (
	"github.com/kairos-io/provider-canonical/pkg/provider"

	"github.com/kairos-io/provider-canonical/pkg/log"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/mudler/go-pluggable"
	"github.com/sirupsen/logrus"
)

func main() {
	log.InitLogger("/var/log/provider-canonical.log")
	logrus.Info("starting provider-canonical")
	plugin := clusterplugin.ClusterPlugin{
		Provider: provider.ClusterProvider,
	}

	if err := plugin.Run(
		pluggable.FactoryPlugin{
			EventType:     clusterplugin.EventClusterReset,
			PluginHandler: provider.HandleClusterReset,
		},
	); err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("completed provider-canonical")
}
