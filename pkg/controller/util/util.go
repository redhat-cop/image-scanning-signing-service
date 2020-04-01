package util

import (
	"errors"
	"strings"
	"time"

	"github.com/redhat-cop/image-security/pkg/controller/images"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	imagev1 "github.com/openshift/api/image/v1"
)

func NewImageExecutionCondition(message string, conditionStatus corev1.ConditionStatus, conditionType images.ImageExecutionConditionType) images.ImageExecutionCondition {

	return images.ImageExecutionCondition{
		LastTransitionTime: metav1.NewTime(time.Now()).String(),
		Message:            message,
		Status:             conditionStatus,
		Type:               conditionType,
	}

}

func ParseImageStreamTag(imageStreamTag string) (string, string) {
	requestIsNameTag := strings.Split(imageStreamTag, ":")

	requestIsName := requestIsNameTag[0]

	var requestIsTag string

	if len(requestIsNameTag) == 2 {
		requestIsTag = requestIsNameTag[1]
	} else {
		requestIsTag = "latest"
	}

	return requestIsName, requestIsTag

}

func LatestTaggedImage(stream *imagev1.ImageStream, tag string) *imagev1.TagEvent {

	// find the most recent tag event with an image reference
	if stream.Status.Tags != nil {
		for _, t := range stream.Status.Tags {
			if t.Tag == tag {
				if len(t.Items) == 0 {
					return nil
				}
				return &t.Items[0]
			}
		}
	}

	return nil

}

func ExtractImageIDFromImageReference(dockerImageReference string) (string, string, error) {

	dockerImageComponents := strings.Split(dockerImageReference, "@")

	if len(dockerImageComponents) != 2 {
		return "", "", errors.New("Unexpected Docker Image Reference")
	}

	dockerImageRegistry := dockerImageComponents[0]
	dockerImageID := dockerImageComponents[1]

	return dockerImageRegistry, dockerImageID, nil
}

func GenerateImageStreamTag(imageStreamTag string, namespace string) *imagev1.ImageStreamTag {
	return &imagev1.ImageStreamTag{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageStreamTag",
			APIVersion: "image.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageStreamTag,
			Namespace: namespace,
		},
	}

}
