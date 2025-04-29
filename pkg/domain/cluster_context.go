package domain

type ClusterContext struct {
	NodeRole               string `json:"nodeRole" yaml:"nodeRole"`
	ClusterCidr            string `json:"clusterCidr" yaml:"clusterCidr"`
	ServiceCidr            string `json:"serviceCidr" yaml:"serviceCidr"`
	ControlPlaneHost       string `json:"controlPlaneHost" yaml:"controlPlaneHost"`
	ClusterToken           string `json:"clusterToken" yaml:"clusterToken"`
	UserOptions            string `json:"userOptions" yaml:"userOptions"`
	LocalImagesPath        string `json:"localImagesPath" yaml:"localImagesPath"`
	CustomAdvertiseAddress string `json:"customAdvertiseAddress" yaml:"customAdvertiseAddress"`

	EnvConfig map[string]string `json:"envConfig" yaml:"envConfig"`
}
