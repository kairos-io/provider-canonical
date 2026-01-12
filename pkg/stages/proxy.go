package stages

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/utils"
	yip "github.com/mudler/yip/pkg/schema"
)

const (
	envFilePrefix = "EnvironmentFile"
	envFilePath   = "/run/provider-canonical/env"
)

// k8sSnapServices lists all snap.k8s services that need proxy drop-in configs.
var k8sSnapServices = []string{
	"snap.k8s.containerd.service",
	"snap.k8s.k8s-apiserver-proxy.service",
	"snap.k8s.k8s-dqlite.service",
	"snap.k8s.k8sd.service",
	"snap.k8s.kube-apiserver.service",
	"snap.k8s.kube-controller-manager.service",
	"snap.k8s.kube-proxy.service",
	"snap.k8s.kube-scheduler.service",
	"snap.k8s.kubelet.service",
	"snap.k8s.etcd.service",
}

// getProxyDropInFiles generates systemd drop-in files for all k8s snap services.
func getProxyDropInFiles() []yip.File {
	content := fmt.Sprintf("[Service]\n%s=-%s", envFilePrefix, envFilePath)
	files := make([]yip.File, 0, len(k8sSnapServices))
	for _, svc := range k8sSnapServices {
		files = append(files, yip.File{
			Path:        filepath.Join("/etc/systemd/system", svc+".d", "http-proxy.conf"),
			Permissions: 0644,
			Content:     content,
		})
	}
	return files
}

func getProviderEnvironmentStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	stages := []yip.Stage{}

	env := map[string]string{}
	if utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		maps.Copy(env, getProxyEnvironments(clusterCtx))
	}

	if len(env) > 0 {
		stages = append(stages, yip.Stage{
			Name:            "Set provider environment",
			Environment:     env,
			EnvironmentFile: envFilePath,
		})
	}

	return stages
}

func getProxyStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	if !utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		return []yip.Stage{}
	}

	files := []yip.File{
		{
			Path:        filepath.Join("/etc/default", "kubelet"),
			Permissions: 0644,
			Content:     kubeletProxyEnv(clusterCtx),
		},
	}
	files = append(files, getProxyDropInFiles()...)

	return []yip.Stage{
		{
			Name:        "Set proxy config files and envs",
			Environment: getProxyEnvironments(clusterCtx),
			Files:       files,
		},
		getProxyServiceReloadStage(),
	}
}

func getProxyServiceReloadStage() yip.Stage {
	commands := []string{"systemctl daemon-reload"}
	for _, svc := range k8sSnapServices {
		commands = append(commands, fmt.Sprintf("systemctl restart %s", svc))
	}
	return yip.Stage{
		Name:     "Reload systemd and restart k8s services after proxy config",
		Commands: commands,
	}
}

func getProxyEnvironments(clusterCtx *domain.ClusterContext) map[string]string {
	proxyEnvs := clusterCtx.EnvConfig

	return map[string]string{
		"HTTP_PROXY":  proxyEnvs["HTTP_PROXY"],
		"HTTPS_PROXY": proxyEnvs["HTTPS_PROXY"],
		"http_proxy":  proxyEnvs["HTTP_PROXY"],
		"https_proxy": proxyEnvs["HTTPS_PROXY"],
		"NO_PROXY":    utils.GetNoProxyConfig(clusterCtx),
		"no_proxy":    utils.GetNoProxyConfig(clusterCtx),
	}
}

func kubeletProxyEnv(clusterCtx *domain.ClusterContext) string {
	var proxy []string

	proxyMap := clusterCtx.EnvConfig

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	noProxy := utils.GetDefaultNoProxy(clusterCtx)
	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
	}

	if len(userNoProxy) > 0 {
		noProxy = noProxy + "," + userNoProxy
	}
	proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
	return strings.Join(proxy, "\n")
}
