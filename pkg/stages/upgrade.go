package stages

import (
	"fmt"
	"path/filepath"

	"github.com/kairos-io/provider-canonical/pkg/domain"
	yip "github.com/mudler/yip/pkg/schema"
)

func getUpgradeStage(clusterCtx *domain.ClusterContext) yip.Stage {
	return yip.Stage{
		Name: "Run Canonical Upgrade",
		Commands: []string{
			fmt.Sprintf("bash %s %s", filepath.Join(domain.CanonicalScriptDir, "upgrade.sh"), clusterCtx.NodeRole),
		},
	}
}
