package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/kairos-io/kairos-sdk/bus"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/mudler/go-pluggable"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func HandleClusterReset(event *pluggable.Event) pluggable.EventResponse {
	logrus.Info("handling cluster reset event")

	var payload bus.EventPayload
	var config clusterplugin.Config
	var response pluggable.EventResponse

	// parse the boot payload
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		logrus.Error("failed to parse reset event: ", err.Error())
		response.Error = fmt.Sprintf("failed to parse reset event: %s", err.Error())
		return response
	}

	// parse config from boot payload
	if err := yaml.Unmarshal([]byte(payload.Config), &config); err != nil {
		logrus.Error("failed to parse config from reset event: ", err.Error())
		response.Error = fmt.Sprintf("failed to parse config from reset event: %s", err.Error())
		return response
	}

	if config.Cluster == nil {
		return response
	}

	cmd := exec.Command(filepath.Join(domain.CanonicalScriptDir, "reset.sh"), string(config.Cluster.Role))
	output, _ := cmd.CombinedOutput()

	logrus.Info("reset node script output: ", string(output))
	return response
}
