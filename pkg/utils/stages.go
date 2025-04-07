package utils

import (
	yip "github.com/mudler/yip/pkg/schema"
)

func GetFileStage(stageName, path, content string, permissions int) yip.Stage {
	return yip.Stage{
		Name: stageName,
		Files: []yip.File{
			{
				Path:        path,
				Permissions: uint32(permissions),
				Content:     content,
			},
		},
	}
}
