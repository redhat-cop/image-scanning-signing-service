package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	imagesigningrequestsv1alpha1 "github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	util "github.com/redhat-cop/image-security/test/e2e"
	"github.com/stretchr/testify/assert"
	kapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCentosSigning(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	util.AddToFrameworkSchemeForTests(t, ctx)
	centosSigning(t, framework.Global, ctx)
}

func centosSigning(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) {
	namespace, err := ctx.GetNamespace()
	assert.NoError(t, err)

	imageSigningRequest := &imagesigningrequestsv1alpha1.ImageSigningRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageSigningRequest",
			APIVersion: "imagesigningrequests.cop.redhat.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.CentosTagName,
			Namespace: util.ImageNamespace,
		},
		Spec: imagesigningrequestsv1alpha1.ImageSigningRequestSpec{
			ContainerImage: &kapi.ObjectReference{
				Kind: "ImageStreamTag",
				Name: "dotnet-example:latest",
			},
		},
	}

	err = f.Client.Create(goctx.TODO(), imageSigningRequest, &framework.CleanupOptions{TestContext: ctx, Timeout: util.Timeout, RetryInterval: util.RetryInterval})
	assert.NoError(t, err)
	assert.Empty(t, err)

	// Check if the CR has been created and has no signed image details
	cr := &imagesigningrequestsv1alpha1.ImageSigningRequest{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: util.CentosTagName, Namespace: util.ImageNamespace}, cr)
	assert.NoError(t, err)
	assert.Empty(t, cr.Status.SignedImage)

	// Check for a pod with the correct annotation
	success := util.WaitForPodWithImageCompleted(t, f, ctx, namespace, "cop.redhat.com/owner", "signing-test/dotnet-app", util.CentosImage, util.RetryInterval, util.Timeout)
	assert.NoError(t, success)

	// Need to wait for the pod controller to pick up the status change of the pod
	err = wait.Poll(util.RetryInterval, util.StatusTimeout, func() (done bool, err error) {
		// Verify the CR has been signing details
		cr := &imagesigningrequestsv1alpha1.ImageSigningRequest{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: util.CentosTagName, Namespace: util.ImageNamespace}, cr)
		if err != nil {
			return false, fmt.Errorf("CR not found")
		}
		if cr.Status.SignedImage != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(t, err)
}
