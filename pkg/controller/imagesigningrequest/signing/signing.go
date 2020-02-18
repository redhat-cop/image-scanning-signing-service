package signing

import (
	"context"
	"os"

	"github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	"github.com/redhat-cop/image-security/pkg/controller/config"
	"github.com/redhat-cop/image-security/pkg/controller/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	corev1 "k8s.io/api/core/v1"
)

func UpdateOnImageSigningCompletionError(message string, imageSigningRequest v1alpha1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := util.NewImageExecutionCondition(message, corev1.ConditionFalse, v1alpha1.ImageExecutionConditionFinished)

	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha1.PhaseFailed)
}

func UpdateOnImageSigningCompletionSuccess(message string, signedImage string, imageSigningRequest v1alpha1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := util.NewImageExecutionCondition(message, corev1.ConditionTrue, v1alpha1.ImageExecutionConditionFinished)

	imageSigningRequestCopy.Status.SignedImage = signedImage
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha1.PhaseCompleted)
}

func UpdateOnImageSigningInitializationFailure(message string, imageSigningRequest v1alpha1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := util.NewImageExecutionCondition(message, corev1.ConditionFalse, v1alpha1.ImageExecutionConditionInitialization)

	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha1.PhaseFailed)
}

func UpdateOnSigningPodLaunch(message string, unsignedImage string, imageSigningRequest v1alpha1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := util.NewImageExecutionCondition(message, corev1.ConditionTrue, v1alpha1.ImageExecutionConditionInitialization)

	imageSigningRequestCopy.Status.UnsignedImage = unsignedImage
	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha1.PhaseRunning)
}

func updateImageSigningRequest(imageSigningRequest *v1alpha1.ImageSigningRequest, condition v1alpha1.ImageExecutionCondition, phase v1alpha1.ImageExecutionPhase) error {

	imageSigningRequest.Status.Conditions = append(imageSigningRequest.Status.Conditions, condition)
	imageSigningRequest.Status.Phase = phase

	err := r.client.Update(context.TODO(), imageScanningRequest)

	return err
}

func LaunchSigningPod(config config.Config, image string, imageDigest string, ownerID string, ownerReference string, gpgSecretName string, gpgSignBy string) (string, error) {

	pod, err := createSigningPod(config.SignScanImage, config.TargetProject, image, imageDigest, ownerID, ownerReference, config.TargetServiceAccount, gpgSecretName, gpgSignBy)

	if err != nil {
		logrus.Errorf("Error Generating Pod: %v'", err)
		return "", err
	}

	err := r.client.Create(context.TODO(), pod)

	if err != nil {
		logrus.Errorf("Error Creating Pod: %v'", err)
		return "", err
	}

	var key string
	if key, err = cache.MetaNamespaceKeyFunc(pod); err != nil {
		runtime.HandleError(err)
		return "", err
	}

	return key, nil
}

func createSigningPod(signScanImage string, targetProject string, image string, imageDigest string, ownerID string, ownerReference string, serviceAccount string, gpgSecret string, signBy string) (*corev1.Pod, error) {

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
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "docker-socket",
						MountPath: "/var/run/docker.sock",
					},
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
					Name: "docker-socket",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/run/docker.sock",
						},
					},
				},
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
	_, sigDemoEnvVal := os.LookupEnv("SIG_DEMO")

	if sigDemoEnvVal {

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

		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "PUSH_TYPE",
			Value: "docker",
		})

		pod.Spec.NodeSelector = map[string]string{"type": "builder"}

	}

	return pod, nil
}