package signing

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	"github.com/redhat-cop/image-security/pkg/controller/config"
	"github.com/redhat-cop/image-security/pkg/controller/images"
	"github.com/redhat-cop/image-security/pkg/controller/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

func UpdateOnImageSigningCompletionError(client client.Client, message string, imageSigningRequest v1alpha1.ImageSigningRequest) error {

	condition := util.NewImageExecutionCondition(message, corev1.ConditionFalse, images.ImageExecutionConditionFinished)

	imageSigningRequest.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(client, &imageSigningRequest, condition, images.PhaseFailed)
}

func UpdateOnImageSigningCompletionSuccess(client client.Client, message string, signedImage string, imageSigningRequest v1alpha1.ImageSigningRequest) error {

	condition := util.NewImageExecutionCondition(message, corev1.ConditionTrue, images.ImageExecutionConditionFinished)

	imageSigningRequest.Status.SignedImage = signedImage
	imageSigningRequest.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(client, &imageSigningRequest, condition, images.PhaseCompleted)
}

func UpdateOnImageSigningInitializationFailure(client client.Client, message string, imageSigningRequest v1alpha1.ImageSigningRequest) error {

	condition := util.NewImageExecutionCondition(message, corev1.ConditionFalse, images.ImageExecutionConditionInitialization)

	imageSigningRequest.Status.StartTime = condition.LastTransitionTime
	imageSigningRequest.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(client, &imageSigningRequest, condition, images.PhaseFailed)
}

func UpdateOnSigningPodLaunch(client client.Client, message string, unsignedImage string, imageSigningRequest v1alpha1.ImageSigningRequest) error {

	condition := util.NewImageExecutionCondition(message, corev1.ConditionTrue, images.ImageExecutionConditionInitialization)

	imageSigningRequest.Status.UnsignedImage = unsignedImage
	imageSigningRequest.Status.StartTime = condition.LastTransitionTime
	imageSigningRequest.Status.EndTime = metav1.NewTime(time.Time{}).String()

	return updateImageSigningRequest(client, &imageSigningRequest, condition, images.PhaseRunning)
}

func updateImageSigningRequest(client client.Client, imageSigningRequest *v1alpha1.ImageSigningRequest, condition images.ImageExecutionCondition, phase images.ImageExecutionPhase) error {

	imageSigningRequest.Status.Conditions = append(imageSigningRequest.Status.Conditions, condition)
	imageSigningRequest.Status.Phase = phase

	err := client.Status().Update(context.TODO(), imageSigningRequest)
	return err
}

func LaunchSigningPod(client client.Client, scheme *runtime.Scheme, config config.Config, instance *v1alpha1.ImageSigningRequest, image string, imageDigest string, ownerID string, ownerReference string, gpgSecretName string, gpgSignBy string) (string, error) {

	pod, err := createSigningPod(scheme, instance, config.SignScanImage, config.TargetProject, image, imageDigest, ownerID, ownerReference, "imagemanager", gpgSecretName, gpgSignBy)
	if err != nil {
		logrus.Errorf("Error Generating Pod: %v'", err)
		return "", err
	}

	err = client.Create(context.TODO(), pod)

	if err != nil {
		logrus.Errorf("Error Creating Pod: %v'", err)
		return "", err
	}

	var key string
	if key, err = cache.MetaNamespaceKeyFunc(pod); err != nil {
		return "", err
	}

	return key, nil
}

func createSigningPod(scheme *runtime.Scheme, instance *v1alpha1.ImageSigningRequest, signScanImage string, targetProject string, image string, imageDigest string, ownerID string, ownerReference string, serviceAccount string, gpgSecret string, signBy string) (*corev1.Pod, error) {
	priv := true
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ownerID,
			Namespace:   targetProject,
			Labels:      map[string]string{"type": "image-signing"},
			Annotations: map[string]string{"cop.redhat.com/owner": ownerReference, "cop.redhat.com/type": "image-signing"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "image-signer",
				Image:           signScanImage,
				ImagePullPolicy: corev1.PullAlways,
				Command:         []string{"/bin/bash", "-c", "mkdir -p ~/.gnupg && cp /root/gpg/* ~/.gnupg && /usr/local/bin/sign-image"},
				Env: []corev1.EnvVar{
					{
						Name:      "NAMESPACE",
						ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
					},
					{
						Name:  "IMAGE",
						Value: image,
					},
					{
						Name:  "PUSH_TYPE",
						Value: "podman",
					},
					{
						Name:  "DIGEST",
						Value: imageDigest,
					},
					{
						Name:  "SIGNBY",
						Value: signBy,
					},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &priv,
				},
				// TODO - Add the containers/sigstore volume
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "gpg",
						MountPath: "/root/gpg",
					},
				},
			}},
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: serviceAccount,
			Volumes: []corev1.Volume{
				{
					Name: "gpg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: gpgSecret,
						},
					},
				},
			},
		},
	}

	// Begin Custom Logic to support signing
	hostPathVal, hostPathBool := os.LookupEnv("HOST_PATH_MOUNT")

	if hostPathBool && strings.EqualFold("true", hostPathVal) {

		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: "sigstore",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/containers/sigstore/",
				},
			},
		})

		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "sigstore",
			MountPath: "/var/lib/containers/sigstore/",
		})

		pod.Spec.NodeSelector = map[string]string{"type": "builder"}

	}

	return pod, nil
}
