package stages

import (
	"fmt"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	yip "github.com/mudler/yip/pkg/schema"
	"path/filepath"
)

func GetPreSetupStages(clusterCtx *domain.ClusterContext) []yip.Stage {
	var stages []yip.Stage
	if proxyStage := getProxyStage(clusterCtx); proxyStage != nil {
		stages = append(stages, *proxyStage)
	}
	return append(stages, getPreCommandStages())
}

func getPreCommandStages() yip.Stage {
	return yip.Stage{
		Name: "Run Pre Setup Commands",
		Commands: []string{
			fmt.Sprintf("/bin/bash %s", filepath.Join(domain.CanonicalScriptDir, "pre-setup.sh")),
		},
	}
}
