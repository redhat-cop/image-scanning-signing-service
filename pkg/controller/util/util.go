package util

import (
	"errors"
	"strings"

	"github.com/redhat-cop/image-security/pkg/controller/images"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	imagev1 "github.com/openshift/api/image/v1"
)

func NewImageExecutionCondition(message string, conditionStatus corev1.ConditionStatus, conditionType images.ImageExecutionConditionType) images.ImageExecutionCondition {

	return images.ImageExecutionCondition{
		LastTransitionTime: metav1.Now(),
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
