package config

import (
	"os"
)

type Config struct {
	TargetProject        string
	SigningTemplate      string
	GpgSecret            string
	GpgSignBy            string
	TargetServiceAccount string
	SignScanImage        string
}

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

func LoadConfig() Config {

	var config Config

	config.TargetProject = getProperty(envTargetProject, defaultTargetProject)

	config.GpgSecret = getProperty(envGpgSecret, defaultGpgSecret)

	config.TargetServiceAccount = getProperty(envTargetServiceAccount, defaultTargetServiceAccount)

	config.GpgSignBy = getProperty(envGpgSignBy, defaultGpgSignBy)

	config.SignScanImage = getProperty(envSignScanImage, defaultSignScanImage)

	return config

}

func getProperty(envProp string, defaultValue string) string {
	value := os.Getenv(envProp)

	if value == "" {
		value = defaultValue
	}

	return value
}
