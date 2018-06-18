package stub

import (
	"context"
	"errors"
	"fmt"
	"strings"

	imagev1 "github.com/openshift/api/image/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/apis/cop/v1alpha2"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/config"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerAgentName = "image-scan-sign-controller"
	ownerAnnotation     = "cop.redhat.com/owner"
)

func NewHandler(config config.Config) sdk.Handler {
	return &Handler{config: config}
}

type Handler struct {
	config config.Config
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha2.ImageSigningRequest:

		if !event.Deleted {

			imageSigningRequest := o
			imageSigningRequestMetadataKey, _ := cache.MetaNamespaceKeyFunc(imageSigningRequest)

			emptyPhase := v1alpha2.ImageSigningRequestStatus{}.Phase
			if imageSigningRequest.Status.Phase == emptyPhase {
				_, requestIsTag := parseImageStreamTag(imageSigningRequest.Spec.ImageStreamTag)

				requestImageStreamTag := &imagev1.ImageStreamTag{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ImageStreamTag",
						APIVersion: "image.openshift.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      imageSigningRequest.Spec.ImageStreamTag,
						Namespace: imageSigningRequest.ObjectMeta.Namespace,
					},
				}

				err := sdk.Get(requestImageStreamTag)

				if err != nil {

					errorMessage := ""

					if k8serrors.IsNotFound(err) {
						errorMessage = fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)
					} else {
						errorMessage = fmt.Sprintf("Error retrieving ImageStreamTag: %v", err)
					}

					logrus.Warnf(errorMessage)
					err = updateOnInitializationFailure(errorMessage, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				}

				dockerImageRegistry, dockerImageID, err := extractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

				if err != nil {
					return err
				}

				if requestImageStreamTag.Image.Signatures != nil {
					errorMessage := fmt.Sprintf("Signatures Exist on Image '%s'", dockerImageID)

					logrus.Warnf(errorMessage)

					err = updateOnInitializationFailure(errorMessage, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				} else {
					logrus.Infof("No Signatures Exist on Image '%s'", dockerImageID)

					// Setup default values
					gpgSecretName := h.config.GpgSecret
					gpgSignBy := h.config.GpgSignBy

					// Check if Secret if found
					if imageSigningRequest.Spec.SigningKeySecretName != "" {

						signingKeySecret := &corev1.Secret{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Secret",
								APIVersion: "v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: imageSigningRequest.Namespace,
								Name:      imageSigningRequest.Spec.SigningKeySecretName,
							},
						}

						err := sdk.Get(signingKeySecret)

						if k8serrors.IsNotFound(err) {

							errorMessage := fmt.Sprintf("GPG Secret '%s' Not Found in Namespace '%s'", imageSigningRequest.Spec.SigningKeySecretName, imageSigningRequest.Namespace)
							logrus.Warnf(errorMessage)
							err = updateOnInitializationFailure(errorMessage, *imageSigningRequest)

							if err != nil {
								return err
							}

							return nil
						}

						logrus.Infof("Copying Secret '%s' to Project '%s'", imageSigningRequest.Spec.SigningKeySecretName, h.config.TargetProject)
						// Create a copy
						signingKeySecretCopy := signingKeySecret.DeepCopy()
						signingKeySecretCopy.Name = string(imageSigningRequest.ObjectMeta.UID)
						signingKeySecretCopy.Namespace = h.config.TargetProject
						signingKeySecretCopy.ResourceVersion = ""
						signingKeySecretCopy.UID = ""

						err = sdk.Create(signingKeySecretCopy)

						if k8serrors.IsAlreadyExists(err) {
							logrus.Info("Secret already exists. Updating...")
							err = sdk.Update(signingKeySecretCopy)
						}

						gpgSecretName = signingKeySecretCopy.Name

						if imageSigningRequest.Spec.SigningKeySignBy != "" {
							gpgSignBy = imageSigningRequest.Spec.SigningKeySignBy
						}

					}

					signingPodName, err := launchPod(h.config, fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), dockerImageID, string(imageSigningRequest.ObjectMeta.UID), imageSigningRequestMetadataKey, gpgSecretName, gpgSignBy)

					if err != nil {
						errorMessage := fmt.Sprintf("Error Occurred Creating Signing Pod '%v'", err)

						logrus.Errorf(errorMessage)

						err = updateOnInitializationFailure(errorMessage, *imageSigningRequest)

						if err != nil {
							return err
						}

						return nil
					}

					logrus.Infof("Signing Pod Launched '%s'", signingPodName)

					err = updateOnSigningPodLaunch(fmt.Sprintf("Signing Pod Launched '%s'", signingPodName), dockerImageID, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				}
			}

		}

	case *corev1.Pod:

		pod := o
		podMetadataKey, _ := cache.MetaNamespaceKeyFunc(pod)

		// Defensive mechanisms
		if pod.ObjectMeta.GetAnnotations() == nil || pod.ObjectMeta.GetAnnotations()[ownerAnnotation] == "" {
			return nil
		}

		podOwnerAnnotation := pod.Annotations[ownerAnnotation]

		isrNamespace, isrName, err := cache.SplitMetaNamespaceKey(podOwnerAnnotation)

		imageSigningRequest := &v1alpha2.ImageSigningRequest{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ImageSigningRequest",
				APIVersion: "cop.redhat.com/v1alpha2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      isrName,
				Namespace: isrNamespace,
			},
		}

		err = sdk.Get(imageSigningRequest)

		if err != nil {
			logrus.Warnf("Could not find ImageSigningRequest '%s' from pod '%s'", podOwnerAnnotation, podMetadataKey)
			return nil
		}

		// Check if ImageSigningRequest has already been marked as Succeeded or Failed
		if imageSigningRequest.Status.Phase == v1alpha2.PhaseCompleted || imageSigningRequest.Status.Phase == v1alpha2.PhaseFailed {
			return nil
		}

		// Check to verfiy ImageSigningRequest is in phase Running
		if imageSigningRequest.Status.Phase != v1alpha2.PhaseRunning {
			return nil
		}

		// Check if Failed
		if pod.Status.Phase == corev1.PodFailed {
			logrus.Infof("Signing Pod Failed. Updating ImageSiginingRequest %s", podOwnerAnnotation)

			err = updateOnCompletionError(fmt.Sprintf("Signing Pod Failed '%v'", err), *imageSigningRequest)

			if err != nil {
				return err
			}

			return nil

		} else if pod.Status.Phase == corev1.PodSucceeded {

			requestImageStreamTag := &imagev1.ImageStreamTag{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ImageStreamTag",
					APIVersion: "image.openshift.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageSigningRequest.Spec.ImageStreamTag,
					Namespace: imageSigningRequest.Namespace,
				},
			}

			err := sdk.Get(requestImageStreamTag)

			if k8serrors.IsNotFound(err) {

				errorMessage := fmt.Sprintf("ImageStream %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)
				logrus.Warnf(errorMessage)

				err = updateOnCompletionError(errorMessage, *imageSigningRequest)

				if err != nil {
					return err
				}

				return nil

			}

			_, dockerImageID, err := extractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

			if err != nil {
				return err
			}

			if requestImageStreamTag.Image.Signatures != nil {

				logrus.Infof("Signing Pod Succeeded. Updating ImageSiginingRequest %s", pod.Annotations[ownerAnnotation])

				err = updateOnCompletionSuccess("Image Signed", dockerImageID, *imageSigningRequest)

				if err != nil {
					return err
				}

			} else {
				err = updateOnCompletionError(fmt.Sprintf("No Signature Exists on Image '%s' After Signing Completed", dockerImageID), *imageSigningRequest)

				if err != nil {
					return err
				}

			}

			return nil
		}

	}
	return nil
}

func launchPod(config config.Config, image string, imageDigest string, ownerID string, ownerReference string, gpgSecretName string, gpgSignBy string) (string, error) {

	pod, err := util.CreateSigningPod(config.SignScanImage, config.TargetProject, image, imageDigest, ownerID, ownerReference, config.TargetServiceAccount, gpgSecretName, gpgSignBy)

	if err != nil {
		logrus.Errorf("Error Generating Pod: %v'", err)
		return "", err
	}

	err = sdk.Create(pod)

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

func updateImageSigningRequest(imageSigningRequest *v1alpha2.ImageSigningRequest, condition v1alpha2.ImageSigningCondition, phase v1alpha2.ImageSigningPhase) error {

	imageSigningRequest.Status.Conditions = append(imageSigningRequest.Status.Conditions, condition)
	imageSigningRequest.Status.Phase = phase

	err := sdk.Update(imageSigningRequest)

	return err
}

func extractImageIDFromImageReference(dockerImageReference string) (string, string, error) {

	dockerImageComponents := strings.Split(dockerImageReference, "@")

	if len(dockerImageComponents) != 2 {
		return "", "", errors.New("Unexpected Docker Image Reference")
	}

	dockerImageRegistry := dockerImageComponents[0]
	dockerImageID := dockerImageComponents[1]

	return dockerImageRegistry, dockerImageID, nil
}

func updateOnSigningPodLaunch(message string, unsignedImage string, imageSigningRequest v1alpha2.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionTrue, v1alpha2.ImageSigningConditionInitialization)

	imageSigningRequestCopy.Status.UnsignedImage = unsignedImage
	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha2.PhaseRunning)
}

func updateOnInitializationFailure(message string, imageSigningRequest v1alpha2.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionFalse, v1alpha2.ImageSigningConditionInitialization)

	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha2.PhaseFailed)
}

func updateOnCompletionError(message string, imageSigningRequest v1alpha2.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionFalse, v1alpha2.ImageSigningConditionFinished)

	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha2.PhaseFailed)
}

func updateOnCompletionSuccess(message string, signedImage string, imageSigningRequest v1alpha2.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionTrue, v1alpha2.ImageSigningConditionFinished)

	imageSigningRequestCopy.Status.SignedImage = signedImage
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return updateImageSigningRequest(imageSigningRequestCopy, condition, v1alpha2.PhaseCompleted)
}

func newImageSigningCondition(message string, conditionStatus corev1.ConditionStatus, conditionType v1alpha2.ImageSigningConditionType) v1alpha2.ImageSigningCondition {

	return v1alpha2.ImageSigningCondition{
		LastTransitionTime: metav1.Now(),
		Message:            message,
		Status:             conditionStatus,
		Type:               conditionType,
	}

}

func parseImageStreamTag(imageStreamTag string) (string, string) {
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

func latestTaggedImage(stream *imagev1.ImageStream, tag string) *imagev1.TagEvent {

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
