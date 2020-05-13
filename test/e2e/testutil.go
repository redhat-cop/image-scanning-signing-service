package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/redhat-cop/image-security/pkg/apis"
	imagesigningrequestsv1alpha1 "github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var (
	RetryInterval        = time.Second * 30
	Timeout              = time.Second * 600
	StatusTimeout        = time.Second * 600
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
	CentosTagName        = "dotnet-app"
	SigningRemoteName    = "signing-app"
	ImageNamespace       = "signing-test" // Namespace that the image to scan exists in
	CentosImage          = "quay.io/cnuland/image-signing-centos8"
)

// Original Source https://github.com/jaegertracing/jaeger-operator/blob/master/test/e2e/utils.go
func GetPod(namespace, key, value, containsImage string, kubeclient kubernetes.Interface) v1.Pod {
	pods, err := kubeclient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return v1.Pod{}
	}
	for _, pod := range pods.Items {
		if strings.Contains(pod.Annotations[key], value) {
			for _, pod := range pods.Items {
				//if strings.HasPrefix(pod.Name, namePrefix) {
				for _, c := range pod.Spec.Containers {
					if strings.Contains(c.Image, containsImage) {
						return pod
					}
					//}
				}
			}
		}
	}
	return v1.Pod{}

}

func WaitForPodWithImageCompleted(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, namespace, key string, value string, image string, retryInterval time.Duration, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		// Check if the CRD has been created
		pod := GetPod(namespace, key, value, image, f.KubeClient)
		if err != nil {
			if apierrors.IsNotFound(err) {
				fmt.Printf("Waiting for availability of pod\n")
				return false, nil
			}
			return false, err
		}
		if pod.Status.Phase == "Succeeded" {
			return true, nil
		}
		fmt.Printf("Waiting for full completion of pod\n")
		return false, nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("pod completed\n")
	return nil
}

func AddToFrameworkSchemeForTests(t *testing.T, ctx *framework.TestCtx) {
	namespace, err := ctx.GetNamespace()
	assert.NoError(t, err)

	imageSigningRequest := &imagesigningrequestsv1alpha1.ImageSigningRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageSigningRequest",
			APIVersion: "imagesigningrequests.cop.redhat.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CentosTagName,
			Namespace: namespace,
		},
		Spec:   imagesigningrequestsv1alpha1.ImageSigningRequestSpec{},
		Status: imagesigningrequestsv1alpha1.ImageSigningRequestStatus{},
	}

	assert.NoError(t, framework.AddToFrameworkScheme(apis.AddToScheme, imageSigningRequest))
}
