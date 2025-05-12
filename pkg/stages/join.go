package stages

import (
	"fmt"

	"path/filepath"

	"gopkg.in/yaml.v3"

	apiv1 "github.com/canonical/k8s-snap-api/api/v1"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	"github.com/kairos-io/provider-canonical/pkg/utils"
	yip "github.com/mudler/yip/pkg/schema"
)

func GetControlPlaneJoinStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	var stages []yip.Stage
	var canonicalConfig apiv1.ControlPlaneJoinConfig
	_ = yaml.Unmarshal([]byte(clusterCtx.UserOptions), &canonicalConfig)

	var bootstrapConfig apiv1.BootstrapConfig
	_ = yaml.Unmarshal([]byte(clusterCtx.UserOptions), &bootstrapConfig)

	allocateNodeCidrs := "true"

	canonicalConfig.ExtraSANS = appendIfNotPresent(canonicalConfig.ExtraSANS, clusterCtx.ControlPlaneHost)
	canonicalConfig.ExtraNodeKubeControllerManagerArgs["--allocate-node-cidrs"] = &allocateNodeCidrs
	canonicalConfig.ExtraNodeKubeControllerManagerArgs["--cluster-cidr"] = bootstrapConfig.PodCIDR

	config, _ := yaml.Marshal(canonicalConfig)

	stages = append(stages,
		getJoinConfigFileStage(string(config)),
		getJoinStage(clusterCtx))

	if dirExists(fs.OSFS, domain.KubeComponentsArgsPath) {
		stages = append(stages, getControlPlaneReconfigureStage(canonicalConfig)...)
	}
	if certStage := getApiserverCertRegenerateStage(canonicalConfig.ExtraSANS); certStage != nil {
		stages = append(stages, *certStage...)
	}
	return stages
}

func GetWorkerJoinStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	var stages []yip.Stage
	var canonicalConfig apiv1.WorkerJoinConfig
	_ = yaml.Unmarshal([]byte(clusterCtx.UserOptions), &canonicalConfig)

	config, _ := yaml.Marshal(canonicalConfig)

	stages = append(stages,
		getJoinConfigFileStage(string(config)),
		getJoinStage(clusterCtx))

	if dirExists(fs.OSFS, domain.KubeComponentsArgsPath) {
		stages = append(stages, getWorkerReconfigureStage(canonicalConfig)...)
	}
	return stages
}

func getJoinConfigFileStage(bootstrapConfig string) yip.Stage {
	return utils.GetFileStage("Generate Join Config", "/opt/canonical/join-config.yaml", bootstrapConfig, 0640)
}

func getJoinStage(clusterCtx *domain.ClusterContext) yip.Stage {
	return yip.Stage{
		Name: "Run Canonical Join",
		If:   fmt.Sprintf("[ ! -f %s ]", "/opt/canonical/canonical.join"),
		Commands: []string{
			fmt.Sprintf("bash %s %s %s %s", filepath.Join(domain.CanonicalScriptDir, "join.sh"), clusterCtx.ClusterToken, clusterCtx.CustomAdvertiseAddress, clusterCtx.NodeRole),
		},
	}
}
