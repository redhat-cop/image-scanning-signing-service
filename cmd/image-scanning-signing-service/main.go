package main

import (
	"context"
	"runtime"

	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/config"
	stub "github.com/redhat-cop/image-scanning-signing-service/pkg/stub"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	configuration := config.LoadConfig()

	sdk.Watch("cop.redhat.com/v1alpha2", "ImageSigningRequest", "", 0)
	sdk.Watch("v1", "Pod", "", 0)
	sdk.Handle(stub.NewHandler(configuration))
	sdk.Run(context.TODO())
}
