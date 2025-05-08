package provider

import (
	apiv1 "github.com/canonical/k8s-snap-api/api/v1"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/stages"
	yip "github.com/mudler/yip/pkg/schema"
	"gopkg.in/yaml.v3"
)

func ClusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	clusterCtx := CreateClusterContext(cluster)

	cfg := yip.YipConfig{
		Name: "Canonical K8s Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": getFinalStages(clusterCtx),
		},
	}

	return cfg
}

func CreateClusterContext(cluster clusterplugin.Cluster) *domain.ClusterContext {
	clusterContext := &domain.ClusterContext{
		NodeRole:         string(cluster.Role),
		EnvConfig:        cluster.Env,
		ControlPlaneHost: cluster.ControlPlaneHost,
		UserOptions:      cluster.Options,
		ClusterToken:     cluster.ClusterToken,
	}

	if address, ok := cluster.ProviderOptions["advertise_address"]; ok && address != "" {
		clusterContext.CustomAdvertiseAddress = address
	} else {
		clusterContext.CustomAdvertiseAddress = "''"
	}

	if cluster.LocalImagesPath == "" {
		clusterContext.LocalImagesPath = domain.DefaultLocalImagesDir
	} else {
		clusterContext.LocalImagesPath = cluster.LocalImagesPath
	}

	return clusterContext
}

func getFinalStages(clusterCtx *domain.ClusterContext) []yip.Stage {
	var finalStages []yip.Stage

	var canonicalConfig apiv1.BootstrapConfig
	_ = yaml.Unmarshal([]byte(clusterCtx.UserOptions), &canonicalConfig)

	setClusterSubnetCtx(clusterCtx, *canonicalConfig.ServiceCIDR, *canonicalConfig.PodCIDR)

	finalStages = append(finalStages, stages.GetPreSetupStages(clusterCtx)...)

	if clusterCtx.NodeRole == clusterplugin.RoleInit {
		finalStages = append(finalStages, stages.GetInitStage(clusterCtx)...)
	} else if clusterCtx.NodeRole == clusterplugin.RoleControlPlane {
		finalStages = append(finalStages, stages.GetControlPlaneJoinStage(clusterCtx)...)
	} else if clusterCtx.NodeRole == clusterplugin.RoleWorker {
		finalStages = append(finalStages, stages.GetWorkerJoinStage(clusterCtx)...)
	}
	return finalStages
}

func setClusterSubnetCtx(clusterCtx *domain.ClusterContext, serviceSubnet, podSubnet string) {
	clusterCtx.ServiceCidr = serviceSubnet
	clusterCtx.ClusterCidr = podSubnet
}
