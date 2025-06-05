package provider

import (
	"testing"

	apiv1 "github.com/canonical/k8s-snap-api/api/v1"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

func TestCreateClusterContext(t *testing.T) {
	g := NewWithT(t)

	t.Run("creates cluster context with default values", func(t *testing.T) {
		cluster := clusterplugin.Cluster{
			Role:             clusterplugin.RoleInit,
			ControlPlaneHost: "10.0.0.1",
			ClusterToken:     "token123",
			Options:          "options",
			Env:              map[string]string{"key": "value"},
		}
		ctx := CreateClusterContext(cluster)

		g.Expect(ctx.NodeRole).To(Equal(string(clusterplugin.RoleInit)))
		g.Expect(ctx.ControlPlaneHost).To(Equal("10.0.0.1"))
		g.Expect(ctx.ClusterToken).To(Equal("token123"))
		g.Expect(ctx.UserOptions).To(Equal("options"))
		g.Expect(ctx.EnvConfig).To(HaveKeyWithValue("key", "value"))
		g.Expect(ctx.CustomAdvertiseAddress).To(Equal("''"))
		g.Expect(ctx.LocalImagesPath).To(Equal(domain.DefaultLocalImagesDir))
	})

	t.Run("sets custom advertise address when provided", func(t *testing.T) {
		cluster := clusterplugin.Cluster{
			ProviderOptions: map[string]string{
				"advertise_address": "192.168.1.1",
			},
		}
		ctx := CreateClusterContext(cluster)
		g.Expect(ctx.CustomAdvertiseAddress).To(Equal("192.168.1.1"))
	})

	t.Run("sets custom local images path when provided", func(t *testing.T) {
		cluster := clusterplugin.Cluster{
			LocalImagesPath: "/custom/path",
		}
		ctx := CreateClusterContext(cluster)
		g.Expect(ctx.LocalImagesPath).To(Equal("/custom/path"))
	})
}

func TestGetFinalStages(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns stages for init role", func(t *testing.T) {
		serviceCIDR := "10.96.0.0/12"
		podCIDR := "10.244.0.0/16"
		profiling := "false"
		config := apiv1.BootstrapConfig{
			ServiceCIDR: &serviceCIDR,
			PodCIDR:     &podCIDR,
			ExtraNodeKubeControllerManagerArgs: map[string]*string{
				"--profiling": &profiling,
			},
		}
		configBytes, err := yaml.Marshal(config)
		g.Expect(err).NotTo(HaveOccurred())

		ctx := &domain.ClusterContext{
			NodeRole:    string(clusterplugin.RoleInit),
			UserOptions: string(configBytes),
		}
		stages := getFinalStages(ctx)

		g.Expect(stages).NotTo(BeEmpty())
		g.Expect(ctx.ServiceCidr).To(Equal(serviceCIDR))
		g.Expect(ctx.ClusterCidr).To(Equal(podCIDR))
	})

	t.Run("returns stages for control plane role", func(t *testing.T) {
		serviceCIDR := "10.96.0.0/12"
		podCIDR := "10.244.0.0/16"
		profiling := "false"
		config := apiv1.BootstrapConfig{
			ServiceCIDR: &serviceCIDR,
			PodCIDR:     &podCIDR,
			ExtraNodeKubeControllerManagerArgs: map[string]*string{
				"--profiling": &profiling,
			},
		}
		configBytes, err := yaml.Marshal(config)
		g.Expect(err).NotTo(HaveOccurred())

		ctx := &domain.ClusterContext{
			NodeRole:    string(clusterplugin.RoleControlPlane),
			UserOptions: string(configBytes),
		}

		stages := getFinalStages(ctx)

		g.Expect(stages).NotTo(BeEmpty())
		g.Expect(ctx.ServiceCidr).To(Equal(serviceCIDR))
		g.Expect(ctx.ClusterCidr).To(Equal(podCIDR))
	})

	t.Run("returns stages for worker role", func(t *testing.T) {
		serviceCIDR := "10.96.0.0/12"
		podCIDR := "10.244.0.0/16"
		config := apiv1.BootstrapConfig{
			ServiceCIDR: &serviceCIDR,
			PodCIDR:     &podCIDR,
		}
		configBytes, err := yaml.Marshal(config)
		g.Expect(err).NotTo(HaveOccurred())

		ctx := &domain.ClusterContext{
			NodeRole:    string(clusterplugin.RoleWorker),
			UserOptions: string(configBytes),
		}

		stages := getFinalStages(ctx)

		g.Expect(stages).NotTo(BeEmpty())
		g.Expect(ctx.ServiceCidr).To(Equal(serviceCIDR))
		g.Expect(ctx.ClusterCidr).To(Equal(podCIDR))
	})
}
