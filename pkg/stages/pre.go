package stages

import (
	"fmt"
	"github.com/kairos-io/provider-canonical/pkg/domain"
	"github.com/kairos-io/provider-canonical/pkg/fs"
	yip "github.com/mudler/yip/pkg/schema"
	"path/filepath"
)

func GetPreSetupStages(clusterCtx *domain.ClusterContext) []yip.Stage {
	var stages []yip.Stage
	if proxyStage := getProxyStage(clusterCtx); proxyStage != nil {
		stages = append(stages, *proxyStage)
	}
	stages = append(stages, getPreCommandStages())
	if dirExists(fs.OSFS, clusterCtx.LocalImagesPath) {
		stages = append(stages, getPreImportLocalImageStage(clusterCtx.LocalImagesPath))
	}
	return stages
}

func getPreCommandStages() yip.Stage {
	return yip.Stage{
		Name: "Run Pre Setup Commands",
		Commands: []string{
			fmt.Sprintf("/bin/bash %s", filepath.Join(domain.CanonicalScriptDir, "pre-setup.sh")),
		},
	}
}

func getPreImportLocalImageStage(localImagesPath string) yip.Stage {
	return yip.Stage{
		Name: "Run Import Local Images",
		Commands: []string{
			fmt.Sprintf("/bin/sh %s %s", filepath.Join(domain.CanonicalScriptDir, "import-images.sh"), localImagesPath),
		},
	}
}
