package stages

import (
	"fmt"
	"testing"

	"github.com/kairos-io/provider-canonical/pkg/domain"
	. "github.com/onsi/gomega"
)

func TestGetProxyDropInFiles(t *testing.T) {
	g := NewWithT(t)

	t.Run("generates drop-in files for all k8s snap services", func(t *testing.T) {
		files := getProxyDropInFiles()

		g.Expect(files).To(HaveLen(len(k8sSnapServices)))

		expectedContent := fmt.Sprintf("[Service]\n%s=-%s", envFilePrefix, envFilePath)
		for i, svc := range k8sSnapServices {
			expectedPath := fmt.Sprintf("/etc/systemd/system/%s.d/http-proxy.conf", svc)
			g.Expect(files[i].Path).To(Equal(expectedPath))
			g.Expect(files[i].Permissions).To(Equal(uint32(0644)))
			g.Expect(files[i].Content).To(Equal(expectedContent))
		}
	})
}

func TestGetProxyServiceReloadStage(t *testing.T) {
	g := NewWithT(t)

	t.Run("generates daemon-reload and restart commands for all services", func(t *testing.T) {
		stage := getProxyServiceReloadStage()

		g.Expect(stage.Name).To(Equal("Reload systemd and restart k8s services after proxy config"))
		g.Expect(stage.Commands[0]).To(Equal("systemctl daemon-reload"))
		g.Expect(stage.Commands).To(HaveLen(len(k8sSnapServices) + 1))

		for i, svc := range k8sSnapServices {
			expectedCmd := fmt.Sprintf("systemctl restart %s", svc)
			g.Expect(stage.Commands[i+1]).To(Equal(expectedCmd))
		}
	})
}

func TestGetProxyStage(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns empty stages when proxy is not configured", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			EnvConfig: map[string]string{},
		}

		stages := getProxyStage(clusterCtx)

		g.Expect(stages).To(BeEmpty())
	})

	t.Run("returns stages with files and environment when proxy is configured", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			ClusterCidr: "10.244.0.0/16",
			ServiceCidr: "10.96.0.0/12",
			EnvConfig: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8443",
			},
		}

		stages := getProxyStage(clusterCtx)

		g.Expect(stages).To(HaveLen(2))
		g.Expect(stages[0].Name).To(Equal("Set proxy config files and envs"))
		g.Expect(stages[1].Name).To(Equal("Reload systemd and restart k8s services after proxy config"))

		// kubelet file + all drop-in files
		expectedFileCount := 1 + len(k8sSnapServices)
		g.Expect(stages[0].Files).To(HaveLen(expectedFileCount))
		g.Expect(stages[0].Files[0].Path).To(Equal("/etc/default/kubelet"))

		// environment variables
		env := stages[0].Environment
		g.Expect(env["HTTP_PROXY"]).To(Equal("http://proxy.example.com:8080"))
		g.Expect(env["HTTPS_PROXY"]).To(Equal("https://proxy.example.com:8443"))
		g.Expect(env["http_proxy"]).To(Equal("http://proxy.example.com:8080"))
		g.Expect(env["https_proxy"]).To(Equal("https://proxy.example.com:8443"))
	})
}

func TestGetProviderEnvironmentStage(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns stage with environment when proxy is configured", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			ClusterCidr: "10.244.0.0/16",
			ServiceCidr: "10.96.0.0/12",
			EnvConfig: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
		}

		stages := getProviderEnvironmentStage(clusterCtx)

		g.Expect(stages).To(HaveLen(1))
		g.Expect(stages[0].Name).To(Equal("Set provider environment"))
		g.Expect(stages[0].EnvironmentFile).To(Equal(envFilePath))
		g.Expect(stages[0].Environment["HTTP_PROXY"]).To(Equal("http://proxy.example.com:8080"))
	})
}

func TestGetProxyEnvironments(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns proxy environment variables with defaults", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			ClusterCidr: "10.244.0.0/16",
			ServiceCidr: "10.96.0.0/12",
			EnvConfig: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://proxy.example.com:8443",
				"NO_PROXY":    "internal.example.com",
			},
		}

		env := getProxyEnvironments(clusterCtx)

		g.Expect(env["HTTP_PROXY"]).To(Equal("http://proxy.example.com:8080"))
		g.Expect(env["HTTPS_PROXY"]).To(Equal("https://proxy.example.com:8443"))
		g.Expect(env["http_proxy"]).To(Equal("http://proxy.example.com:8080"))
		g.Expect(env["https_proxy"]).To(Equal("https://proxy.example.com:8443"))

		// includes user and default NO_PROXY values
		g.Expect(env["NO_PROXY"]).To(ContainSubstring("internal.example.com"))
		g.Expect(env["NO_PROXY"]).To(ContainSubstring("10.244.0.0/16"))
		g.Expect(env["NO_PROXY"]).To(ContainSubstring(".svc"))
	})
}

func TestKubeletProxyEnv(t *testing.T) {
	g := NewWithT(t)

	t.Run("generates kubelet environment with proxy settings", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			ClusterCidr: "10.244.0.0/16",
			ServiceCidr: "10.96.0.0/12",
			EnvConfig: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
				"NO_PROXY":   "custom.internal.com",
			},
		}

		content := kubeletProxyEnv(clusterCtx)

		g.Expect(content).To(ContainSubstring("HTTP_PROXY=http://proxy.example.com:8080"))
		g.Expect(content).To(ContainSubstring("NO_PROXY="))
		g.Expect(content).To(ContainSubstring("custom.internal.com"))
	})

	t.Run("always includes NO_PROXY even without user config", func(t *testing.T) {
		clusterCtx := &domain.ClusterContext{
			ClusterCidr: "10.244.0.0/16",
			ServiceCidr: "10.96.0.0/12",
			EnvConfig:   map[string]string{},
		}

		content := kubeletProxyEnv(clusterCtx)

		g.Expect(content).To(ContainSubstring("NO_PROXY="))
	})
}
