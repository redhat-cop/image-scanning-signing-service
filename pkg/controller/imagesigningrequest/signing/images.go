package signing

import (
	"errors"
	"strings"

	imageset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	kapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetImageLocationFromRequest(client *imageset.ImageV1Client, image *kapi.ObjectReference, namespace string) (string, string, error) {
	if image != nil && (image.Kind == "ImageStreamImage") {
		dockerImageComponents := strings.Split(image.Name, "@")

		if len(dockerImageComponents) != 2 {
			return "", "", errors.New("Unexpected ImageStreamImage Reference")
		}

		return image.Name, dockerImageComponents[1], nil
	}

	if image != nil && (image.Kind == "ImageStreamTag") {
		requestImageStreamTag, err := client.ImageStreamTags(namespace).Get(image.Name, metav1.GetOptions{})
		dockerImageComponents := strings.Split(requestImageStreamTag.Image.DockerImageReference, "@")

		if len(dockerImageComponents) != 2 {
			return "", "", errors.New("Unexpected ImageStreamTag Reference")
		}

		if err != nil {
			return "", "", errors.New("Error finding ImageStreamTag")
		}

		return requestImageStreamTag.Image.DockerImageReference, dockerImageComponents[1], nil

	}
	if image != nil && (image.Kind == "DockerImage") {
		dockerImageComponents := strings.Split(image.Name, ":")

		return image.Name, dockerImageComponents[0], nil
	}

	return "", "", errors.New("Unexpected Image Reference")
}
