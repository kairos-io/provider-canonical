package stages

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/kairos-io/provider-canonical/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/twpayne/go-vfs/v4"

	apiv1 "github.com/canonical/k8s-snap-api/api/v1"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	yip "github.com/mudler/yip/pkg/schema"
)

func getBootstrapReconfigureStage(config apiv1.BootstrapConfig) []yip.Stage {
	return getReconfigureStage(config.ExtraNodeKubeAPIServerArgs, config.ExtraNodeKubeControllerManagerArgs,
		config.ExtraNodeKubeSchedulerArgs, config.ExtraNodeKubeProxyArgs, config.ExtraNodeKubeletArgs)
}

func getControlPlaneReconfigureStage(config apiv1.ControlPlaneJoinConfig) []yip.Stage {
	return getReconfigureStage(config.ExtraNodeKubeAPIServerArgs, config.ExtraNodeKubeControllerManagerArgs,
		config.ExtraNodeKubeSchedulerArgs, config.ExtraNodeKubeProxyArgs, config.ExtraNodeKubeletArgs)
}

func getReconfigureStage(apiserver, controller, scheduler, kubeProxy, kubelet map[string]*string) []yip.Stage {
	return []yip.Stage{
		getReconfigureFileStage(apiserver, controller, scheduler, kubeProxy, kubelet),
		getReconfigureServiceRestartStage(),
	}
}

func getReconfigureFileStage(apiserver, controller, scheduler, kubeProxy, kubelet map[string]*string) yip.Stage {
	return yip.Stage{
		Name: "Regenerate Kube Components Args Files",
		Files: []yip.File{
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kube-apiserver"),
				Permissions: 0600,
				Content:     getApiserverArgs(apiserver),
			},
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kube-controller-manager"),
				Permissions: 0600,
				Content:     getKubeControllerArgs(controller),
			},
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kube-scheduler"),
				Permissions: 0600,
				Content:     getKubeSchedulerArgs(scheduler),
			},
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kube-proxy"),
				Permissions: 0600,
				Content:     getKubeProxyArgs(kubeProxy),
			},
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kubelet"),
				Permissions: 0600,
				Content:     getKubeletArgs(kubelet),
			},
		},
	}
}

func getReconfigureServiceRestartStage() yip.Stage {
	return yip.Stage{
		Name: "Restart Kube Components Services",
		Commands: []string{
			"systemctl daemon-reload",
			"systemctl restart snap.k8s.kube-apiserver.service",
			"systemctl restart snap.k8s.kube-controller-manager.service",
			"systemctl restart snap.k8s.kube-scheduler.service",
			"systemctl restart snap.k8s.kube-proxy.service",
			"systemctl restart snap.k8s.kubelet.service",
		},
	}
}

func getWorkerReconfigureStage(canonicalConfig apiv1.WorkerJoinConfig) []yip.Stage {
	return []yip.Stage{
		getWorkerReconfigureFileStage(canonicalConfig),
		getWorkerReconfigureServiceRestartStage(),
	}
}

func getWorkerReconfigureFileStage(canonicalConfig apiv1.WorkerJoinConfig) yip.Stage {
	return yip.Stage{
		Name: "Regenerate Kube Components Args Files",
		Files: []yip.File{
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kube-proxy"),
				Permissions: 0600,
				Content:     getKubeProxyArgs(canonicalConfig.ExtraNodeKubeProxyArgs),
			},
			{
				Path:        filepath.Join(domain.KubeComponentsArgsPath, "kubelet"),
				Permissions: 0600,
				Content:     getKubeletArgs(canonicalConfig.ExtraNodeKubeletArgs),
			},
		},
	}
}

func getWorkerReconfigureServiceRestartStage() yip.Stage {
	return yip.Stage{
		Name: "Restart Kube Components Services",
		Commands: []string{
			"systemctl daemon-reload",
			"systemctl restart snap.k8s.kube-proxy.service",
			"systemctl restart snap.k8s.kubelet.service",
		},
	}
}

func getCertRegenerateStage(incomingSans []string) *yip.Stage {
	if len(incomingSans) == 0 {
		return nil
	}

	apiserverCertPath := filepath.Join(domain.KubeCertificateDirPath, "apiserver.crt")
	if !utils.FileExists(fs.OSFS, apiserverCertPath) {
		return nil
	}

	existingSans, err := utils.GetCertSans(apiserverCertPath)
	if err != nil {
		logrus.Fatalf("failed to get cert sans: %v", err)
	}
	if containsAnyNonMatch(incomingSans, existingSans) {
		cmd := fmt.Sprintf("k8s refresh-certs --expires-in 20y %s", generateExtraSANSString(incomingSans))
		return &yip.Stage{
			Name:     "Regenerate Certificates",
			Commands: []string{cmd},
		}
	}
	return nil
}

func generateExtraSANSString(sans []string) string {
	var result strings.Builder

	for i, san := range sans {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString("--extra-sans ")
		result.WriteString(san)
	}
	return result.String()
}

func getApiserverArgs(updatedArgs map[string]*string) string {
	return getArgs(updatedArgs, "kube-apiserver")
}

func getKubeControllerArgs(updatedArgs map[string]*string) string {
	return getArgs(updatedArgs, "kube-controller-manager")
}

func getKubeSchedulerArgs(updatedArgs map[string]*string) string {
	return getArgs(updatedArgs, "kube-scheduler")
}

func getKubeProxyArgs(updatedArgs map[string]*string) string {
	return getArgs(updatedArgs, "kube-proxy")
}

func getKubeletArgs(updatedArgs map[string]*string) string {
	return getArgs(updatedArgs, "kubelet")
}

func getArgs(updatedArgs map[string]*string, serviceName string) string {
	currentArgs, _ := readServiceArgsFile(fs.OSFS, serviceName)
	maps.Copy(currentArgs, updatedArgs)

	var args []string
	for key, value := range currentArgs {
		args = append(args, fmt.Sprintf("%s=%v", key, *value))
	}
	return strings.Join(args, "\n")
}

func readServiceArgsFile(root vfs.FS, serviceName string) (map[string]*string, error) {
	file, err := root.OpenFile(filepath.Join(domain.KubeComponentsArgsPath, serviceName), os.O_RDONLY, 0600)
	if err != nil {
		logrus.Fatal(err)
	}
	defer file.Close()

	args := make(map[string]*string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			args[key] = &value
		}
	}

	if err = scanner.Err(); err != nil {
		logrus.Fatal(err)
	}
	return args, nil
}

func containsAnyNonMatch(sources []string, targets []string) bool {
	for _, source := range sources {
		found := false
		for _, target := range targets {
			if source == target {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}
