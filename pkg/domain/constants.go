package domain

const (
	K8sNoProxy             = ".svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1"
	KubeComponentsArgsPath = "/var/snap/k8s/common/args"
	KubeCertificateDirPath = "/etc/kubernetes/pki"

	CanonicalScriptDir    = "/opt/canonical/scripts"
	DefaultLocalImagesDir = "/opt/canonical/images"
)
