package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateSigningPod(signScanImage string, image string, imageDigest string, ownerID string, ownerReference string, serviceAccount string, gpgSecret string, signBy string) (*corev1.Pod, error) {

	priv := true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ownerID,
			Labels:      map[string]string{"type": "image-signing"},
			Annotations: map[string]string{"cop.redhat.com/owner": ownerReference},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "image-signer",
				Image:           signScanImage,
				ImagePullPolicy: corev1.PullAlways,
				Command:         []string{"/usr/local/bin/sign-image"},
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
						MountPath: "/root/.gnupg",
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

	return pod, nil
}
