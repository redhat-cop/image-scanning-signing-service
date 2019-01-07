package stub

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/apis/cop/v1alpha2"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/common"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/config"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/openscap"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/scanning"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/signing"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/util"
	"k8s.io/apimachinery/pkg/util/net"

	"encoding/xml"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
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
				_, requestIsTag := util.ParseImageStreamTag(imageSigningRequest.Spec.ImageStreamTag)

				requestImageStreamTag := util.GenerateImageStreamTag(imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.ObjectMeta.Namespace)

				err := sdk.Get(requestImageStreamTag)

				if err != nil {

					errorMessage := ""

					if k8serrors.IsNotFound(err) {
						errorMessage = fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)
					} else {
						errorMessage = fmt.Sprintf("Error retrieving ImageStreamTag: %v", err)
					}

					logrus.Warnf(errorMessage)
					err = signing.UpdateOnImageSigningInitializationFailure(errorMessage, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				}

				dockerImageRegistry, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

				if err != nil {
					return err
				}

				if requestImageStreamTag.Image.Signatures != nil {
					errorMessage := fmt.Sprintf("Signatures Exist on Image '%s'", dockerImageID)

					logrus.Warnf(errorMessage)

					err = signing.UpdateOnImageSigningInitializationFailure(errorMessage, *imageSigningRequest)

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
							err = signing.UpdateOnImageSigningInitializationFailure(errorMessage, *imageSigningRequest)

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

					signingPodName, err := signing.LaunchSigningPod(h.config, fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), dockerImageID, string(imageSigningRequest.ObjectMeta.UID), imageSigningRequestMetadataKey, gpgSecretName, gpgSignBy)

					if err != nil {
						errorMessage := fmt.Sprintf("Error Occurred Creating Signing Pod '%v'", err)

						logrus.Errorf(errorMessage)

						err = signing.UpdateOnImageSigningInitializationFailure(errorMessage, *imageSigningRequest)

						if err != nil {
							return err
						}

						return nil
					}

					logrus.Infof("Signing Pod Launched '%s'", signingPodName)

					err = signing.UpdateOnSigningPodLaunch(fmt.Sprintf("Signing Pod Launched '%s'", signingPodName), dockerImageID, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				}
			}

		}

	case *v1alpha2.ImageScanningRequest:
		if !event.Deleted {

			imageScanningRequest := o
			imageScanningRequestMetadataKey, _ := cache.MetaNamespaceKeyFunc(imageScanningRequest)

			emptyPhase := v1alpha2.ImageScanningRequestStatus{}.Phase
			if imageScanningRequest.Status.Phase == emptyPhase {
				_, requestIsTag := util.ParseImageStreamTag(imageScanningRequest.Spec.ImageStreamTag)

				requestImageStreamTag := util.GenerateImageStreamTag(imageScanningRequest.Spec.ImageStreamTag, imageScanningRequest.ObjectMeta.Namespace)

				err := sdk.Get(requestImageStreamTag)

				if err != nil {

					errorMessage := ""

					if k8serrors.IsNotFound(err) {
						errorMessage = fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", imageScanningRequest.Spec.ImageStreamTag, imageScanningRequest.Namespace)
					} else {
						errorMessage = fmt.Sprintf("Error retrieving ImageStreamTag: %v", err)
					}

					logrus.Warnf(errorMessage)
					err = scanning.UpdateOnImageScanningInitializationFailure(errorMessage, *imageScanningRequest)

					if err != nil {
						return err
					}

					return nil

				}

				dockerImageRegistry, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

				if err != nil {
					return err
				}

				scanningPodName, err := scanning.LaunchScanningPod(h.config, fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), string(imageScanningRequest.ObjectMeta.UID), imageScanningRequestMetadataKey)

				if err != nil {
					errorMessage := fmt.Sprintf("Error Occurred Creating Scanning Pod '%v'", err)

					logrus.Errorf(errorMessage)

					err = scanning.UpdateOnImageScanningInitializationFailure(errorMessage, *imageScanningRequest)

					if err != nil {
						return err
					}

					return nil
				}

				logrus.Infof("Scanning Pod Launched '%s'", scanningPodName)

				err = scanning.UpdateOnScanningPodLaunch(fmt.Sprintf("Scanning Pod Launched '%s'", scanningPodName), dockerImageID, *imageScanningRequest)

				if err != nil {
					return err
				}

				return nil

			}

		}

	case *corev1.Pod:

		pod := o
		podMetadataKey, _ := cache.MetaNamespaceKeyFunc(pod)

		podNamespace, podName, _ := cache.SplitMetaNamespaceKey(podMetadataKey)

		// Defensive mechanisms
		if pod.ObjectMeta.GetAnnotations() == nil || pod.ObjectMeta.GetAnnotations()[common.CopOwnerAnnotation] == "" || pod.ObjectMeta.GetAnnotations()[common.CopTypeAnnotation] == "" {
			return nil
		}

		podTypeAnnotation := pod.Annotations[common.CopTypeAnnotation]

		if podTypeAnnotation == common.ImageSigningTypeAnnotation {

			podOwnerAnnotation := pod.Annotations[common.CopOwnerAnnotation]

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

				err = signing.UpdateOnImageSigningCompletionError(fmt.Sprintf("Signing Pod Failed '%v'", err), *imageSigningRequest)

				if err != nil {
					return err
				}

				return nil

			} else if pod.Status.Phase == corev1.PodSucceeded {

				requestImageStreamTag := util.GenerateImageStreamTag(imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)

				err := sdk.Get(requestImageStreamTag)

				if k8serrors.IsNotFound(err) {

					errorMessage := fmt.Sprintf("ImageStream %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)
					logrus.Warnf(errorMessage)

					err = signing.UpdateOnImageSigningCompletionError(errorMessage, *imageSigningRequest)

					if err != nil {
						return err
					}

					return nil

				}

				_, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

				if err != nil {
					return err
				}

				if requestImageStreamTag.Image.Signatures != nil {

					logrus.Infof("Signing Pod Succeeded. Updating ImageSiginingRequest %s", pod.Annotations[common.CopOwnerAnnotation])

					err = signing.UpdateOnImageSigningCompletionSuccess("Image Signed", dockerImageID, *imageSigningRequest)

					if err != nil {
						return err
					}

				} else {
					err = signing.UpdateOnImageSigningCompletionError(fmt.Sprintf("No Signature Exists on Image '%s' After Signing Completed", dockerImageID), *imageSigningRequest)

					if err != nil {
						return err
					}

				}

				return nil
			}
		} else if podTypeAnnotation == common.ImageScanningTypeAnnotation {
			podOwnerAnnotation := pod.Annotations[common.CopOwnerAnnotation]

			isrNamespace, isrName, err := cache.SplitMetaNamespaceKey(podOwnerAnnotation)

			imageScanningRequest := &v1alpha2.ImageScanningRequest{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ImageScanningRequest",
					APIVersion: "cop.redhat.com/v1alpha2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      isrName,
					Namespace: isrNamespace,
				},
			}

			err = sdk.Get(imageScanningRequest)

			if err != nil {
				logrus.Warnf("Could not find ImageScanningRequest '%s' from pod '%s'", podOwnerAnnotation, podMetadataKey)
				return nil
			}

			// Check if ImageScanningRequest has already been marked as Succeeded or Failed
			if imageScanningRequest.Status.Phase == v1alpha2.PhaseCompleted || imageScanningRequest.Status.Phase == v1alpha2.PhaseFailed {
				return nil
			}

			// Check to verfiy ImageSigningRequest is in phase Running
			if imageScanningRequest.Status.Phase != v1alpha2.PhaseRunning {
				return nil
			}

			// Check if Failed
			if pod.Status.Phase == corev1.PodFailed {
				logrus.Infof("Scanning Pod Failed. Updating ImageSiginingRequest %s", podOwnerAnnotation)

				err = scanning.UpdateOnImageScanningCompletionError(fmt.Sprintf("Scanning Pod Failed '%v'", err), *imageScanningRequest)

				if err != nil {
					return err
				}

				return nil

			} else if pod.Status.Phase == corev1.PodRunning {
				logrus.Infof("Scanning Pod '%s' in Namespace '%s' is Running. Waiting for Scan to Complete...", podName, podNamespace)

				var delay = time.Duration(10) * time.Second
				attempts := 20

				c1 := make(chan bool)

				go func() {

					for i := 1; i <= attempts; i++ {

						healthRequest := k8sclient.GetKubeClient().Core().RESTClient().Get().
							Namespace(podNamespace).
							Resource("pods").
							SubResource("proxy").
							Name(net.JoinSchemeNamePort("http", podName, "8080")).
							Suffix("healthz")

						result := healthRequest.Do()
						var statusCode int
						result.StatusCode(&statusCode)

						if http.StatusOK == statusCode {
							c1 <- true
							return
						}
						time.Sleep(delay)
					}

					c1 <- false
				}()

				podHealthy := <-c1

				if podHealthy {

					logrus.Infoln("Retrieving OpenSCAP Report")

					openSCAPReportResult := k8sclient.GetKubeClient().Core().RESTClient().Get().
						Namespace(podNamespace).
						Resource("pods").
						SubResource("proxy").
						Name(net.JoinSchemeNamePort("http", podName, "8080")).
						Suffix("/api/v1/openscap").Do()

					failedRules, passedRules, totalRules := 0, 0, 0

					var openSCAPReport openscap.OpenSCAPReport

					resp, err := openSCAPReportResult.Raw()

					if err != nil {
						logrus.Errorf("Failed to Retrieve OpenScap Report %v", err)

						err = scanning.UpdateOnImageScanningCompletionError("OpenSCAP Report Retrieval Failure", *imageScanningRequest)

						if err != nil {
							return err
						}

						return nil
					}

					err = xml.Unmarshal(resp, &openSCAPReport)

					if err != nil {
						logrus.Errorf("Failed Unmarshalling OpenSCAP Report %v", err)

						err = scanning.UpdateOnImageScanningCompletionError("Failed Unmarshalling OpenSCAP Report", *imageScanningRequest)

						if err != nil {
							return err
						}

						return nil
					}

					for i := 0; i < len(openSCAPReport.Reports); i++ {
						for j := 0; j < len(openSCAPReport.Reports[i].Report[i].Content.TestResult.RuleResults); j++ {
							result := openSCAPReport.Reports[i].Report[i].Content.TestResult.RuleResults[j].Result

							if result == "pass" {
								passedRules++

							} else if result == "fail" {
								failedRules++
							}

							totalRules++

						}
					}

					logrus.Infof("Scanning Pod Succeeded. Updating ImageScanningRequest %s", pod.Annotations[common.CopOwnerAnnotation])

					err = scanning.UpdateOnImageScanningCompletionSuccess("Image Scanned", totalRules, passedRules, failedRules, *imageScanningRequest)

					if err != nil {
						return err
					}

					// Best Effort Delete Scanning Pod
					err = scanning.DeleteScanningPod(podName, podNamespace)

					if err != nil {
						logrus.Warnf("Failed to Delete Scanning Pod '%s': %v", podName, err)
					}

				} else {
					logrus.Infof("Scanning Health Check Could Not Be Validated. Updating ImageScanningRequest %s", podOwnerAnnotation)

					err = scanning.UpdateOnImageScanningCompletionError("Health Check Validation Error", *imageScanningRequest)

					if err != nil {
						return err
					}

					// Best Effort Delete Scanning Pod
					err = scanning.DeleteScanningPod(podName, podNamespace)

					if err != nil {
						logrus.Warnf("Failed to Delete Scanning Pod '%s': %v", podName, err)
					}

					return nil

				}

			}
		}

	}
	return nil
}
