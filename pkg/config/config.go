package config

import (
	"flag"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	K8SConfig            rest.Config
	TargetProject        string
	SigningTemplate      string
	GpgSecret            string
	GpgSignBy            string
	TargetServiceAccount string
	SignScanImage        string
}

// optional - local kubeconfig for testing
var kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig file")

const (
	defaultTargetProject        = "image-management"
	envTargetProject            = "TARGET_PROJECT"
	defaultTargetServiceAccount = "imagemanager"
	envTargetServiceAccount     = "TARGET_SERVICE_ACCOUNT"
	defaultGpgSecret            = "gpg"
	envGpgSecret                = "GPG_SECRET"
	defaultGpgSignBy            = "openshift@example.com"
	envGpgSignBy                = "GPG_SIGN_BY"
	defaultSignScanImage        = "image-sign-scan-base"
	envSignScanImage            = "SIGN_SCAN_IMAGE"
)

func LoadConfig() (Config, error) {

	var config Config

	// send logs to stderr so we can use 'kubectl logs'
	flag.Set("logtostderr", "true")
	flag.Set("v", "3")
	flag.Parse()

	restConfig, error := getConfig(*kubeconfig)

	if error != nil {
		return config, error
	}

	config.K8SConfig = *restConfig

	config.TargetProject = getProperty(envTargetProject, defaultTargetProject)

	config.GpgSecret = getProperty(envGpgSecret, defaultGpgSecret)

	config.TargetServiceAccount = getProperty(envTargetServiceAccount, defaultTargetServiceAccount)

	config.GpgSignBy = getProperty(envGpgSignBy, defaultGpgSignBy)

	config.SignScanImage = getProperty(envSignScanImage, defaultSignScanImage)

	return config, error

}
func getConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func getProperty(envProp string, defaultValue string) string {
	value := os.Getenv(envProp)

	if value == "" {
		value = defaultValue
	}

	return value
}
