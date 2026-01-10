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
	envPrefix = "Environment="
)

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
			EnvironmentFile: "/run/provider-canonical/env",
		})
	}

	return stages
}

func getProxyStage(clusterCtx *domain.ClusterContext) []yip.Stage {
	if !utils.IsProxyConfigured(clusterCtx.EnvConfig) {
		return []yip.Stage{}
	}

	return []yip.Stage{
		{
			Name:        "Set proxy config files and envs",
			Environment: getProxyEnvironments(clusterCtx),
			Files: []yip.File{
				{
					Path:        filepath.Join("/etc/default", "kubelet"),
					Permissions: 0644,
					Content:     kubeletProxyEnv(clusterCtx),
				},
				{
					Path:        filepath.Join("/etc/systemd/system/snap.k8s.containerd.service.d", "http-proxy.conf"),
					Permissions: 0644,
					Content:     containerdProxyEnv(clusterCtx),
				},
			},
		},
		getProxyServiceReloadStage(),
	}
}

func getProxyServiceReloadStage() yip.Stage {
	return yip.Stage{
		Name: "Reload systemd and restart containerd after proxy config",
		Commands: []string{
			"systemctl daemon-reload",
			"systemctl restart snap.k8s.containerd.service",
		},
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

func containerdProxyEnv(clusterCtx *domain.ClusterContext) string {
	var proxy []string

	proxyMap := clusterCtx.EnvConfig

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	proxy = append(proxy, "[Service]")
	noProxy := utils.GetDefaultNoProxy(clusterCtx)

	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTP_PROXY=%s"+"\"", httpProxy))
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"http_proxy=%s"+"\"", httpProxy))
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"HTTPS_PROXY=%s"+"\"", httpsProxy))
		proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"https_proxy=%s"+"\"", httpProxy))
	}

	if len(userNoProxy) > 0 {
		noProxy = noProxy + "," + userNoProxy
	}
	proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"NO_PROXY=%s"+"\"", noProxy))
	proxy = append(proxy, fmt.Sprintf(envPrefix+"\""+"no_proxy=%s"+"\"", noProxy))

	return strings.Join(proxy, "\n")
}
