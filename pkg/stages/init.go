package stages

import (
	"fmt"
	"path/filepath"

	apiv1 "github.com/canonical/k8s-snap-api/api/v1"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	"github.com/kairos-io/provider-canonical/pkg/utils"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/twpayne/go-vfs/v4"
	"gopkg.in/yaml.v3"
)

func GetInitStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	var stages []yip.Stage
	var canonicalConfig apiv1.BootstrapConfig
	_ = yaml.Unmarshal([]byte(clusterCtx.UserOptions), &canonicalConfig)

	canonicalConfig.ExtraSANs = appendIfNotPresent(canonicalConfig.ExtraSANs, clusterCtx.ControlPlaneHost)

	enableDns := true
	allocateNodeCidrs := "true"

	canonicalConfig.ClusterConfig.DNS.Enabled = &enableDns
	canonicalConfig.ExtraNodeKubeControllerManagerArgs["--allocate-node-cidrs"] = &allocateNodeCidrs
	canonicalConfig.ExtraNodeKubeControllerManagerArgs["--cluster-cidr"] = canonicalConfig.PodCIDR

	config, _ := yaml.Marshal(canonicalConfig)

	stages = append(stages,
		getConfigFileStage(string(config)),
		getBootstrapStage(clusterCtx.CustomAdvertiseAddress))

	if dirExists(fs.OSFS, domain.KubeComponentsArgsPath) {
		stages = append(stages, getBootstrapReconfigureStage(canonicalConfig)...)
	}

	if certStage := getApiserverCertRegenerateStage(canonicalConfig.ExtraSANs); certStage != nil {
		stages = append(stages, *certStage...)
	}

	return stages
}

func getConfigFileStage(bootstrapConfig string) yip.Stage {
	return utils.GetFileStage("Generate Bootstrap Config", "/opt/canonical/bootstrap-config.yaml", bootstrapConfig, 0640)
}

func getBootstrapStage(advertiseAddress string) yip.Stage {
	return yip.Stage{
		Name: "Run Canonical Bootstrap",
		If:   fmt.Sprintf("[ ! -f %s ]", "/opt/canonical/canonical.bootstrap"),
		Commands: []string{
			fmt.Sprintf("bash %s %s", filepath.Join(domain.CanonicalScriptDir, "bootstrap.sh"), advertiseAddress),
		},
	}
}

func dirExists(fs vfs.FS, path string) bool {
	info, err := fs.Stat(path)
	return err == nil && info.IsDir()
}

func appendIfNotPresent(slice []string, element string) []string {
	for _, e := range slice {
		if e == element {
			return slice
		}
	}
	return append(slice, element)
}
