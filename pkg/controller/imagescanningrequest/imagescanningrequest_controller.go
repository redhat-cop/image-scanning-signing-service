package imagescanningrequest

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	imagev1 "github.com/openshift/api/image/v1"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/common"
	imagescanningrequestsv1alpha1 "github.com/redhat-cop/image-security/pkg/apis/imagescanningrequests/v1alpha1"
	"github.com/redhat-cop/image-security/pkg/controller/config"
	"github.com/redhat-cop/image-security/pkg/controller/images"
	"github.com/redhat-cop/image-security/pkg/controller/imagescanningrequest/openscap"
	"github.com/redhat-cop/image-security/pkg/controller/imagescanningrequest/scanning"
	"github.com/redhat-cop/image-security/pkg/controller/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_imagescanningrequest")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ImageScanningRequest Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	k8sclient, err := kubernetes.NewForConfig(mgr.GetConfig())

	if err != nil {
		return err
	}

	return add(mgr, newReconciler(mgr, k8sclient))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sclient kubernetes.Interface) reconcile.Reconciler {
	configuration := config.LoadConfig()
	return &ReconcileImageScanningRequest{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: configuration}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagescanningrequest-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ImageScanningRequest
	err = c.Watch(&source.Kind{Type: &imagescanningrequestsv1alpha1.ImageScanningRequest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ImageScanningRequest
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &imagescanningrequestsv1alpha1.ImageScanningRequest{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileImageScanningRequest implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileImageScanningRequest{}

// ReconcileImageScanningRequest reconciles a ImageScanningRequest object
type ReconcileImageScanningRequest struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	scheme    *runtime.Scheme
	config    config.Config
	k8sclient kubernetes.Interface
}

// Reconcile reads that state of the cluster for a ImageScanningRequest object and makes changes based on the state read
// and what is in the ImageScanningRequest.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileImageScanningRequest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ImageScanningRequest")

	// Fetch the ImageScanningRequest instance
	instance := &imagescanningrequestsv1alpha1.ImageScanningRequest{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	imageScanningRequestMetadataKey, _ := cache.MetaNamespaceKeyFunc(instance)

	emptyPhase := imagescanningrequestsv1alpha1.ImageScanningRequestStatus{}.Phase
	if instance.Status.Phase == emptyPhase {
		_, requestIsTag := util.ParseImageStreamTag(instance.Spec.ImageStreamTag)

		requestImageStreamTag := &imagev1.ImageStreamTag{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ImageStreamTag, Namespace: instance.ObjectMeta.Namespace}, requestImageStreamTag)

		if err != nil {

			errorMessage := ""

			if k8serrors.IsNotFound(err) {
				errorMessage = fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", instance.Spec.ImageStreamTag, instance.Namespace)
			} else {
				errorMessage = fmt.Sprintf("Error retrieving ImageStreamTag: %v", err)
			}

			logrus.Warnf(errorMessage)
			err = scanning.UpdateOnImageScanningInitializationFailure(r.client, errorMessage, *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil

		}

		dockerImageRegistry, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

		if err != nil {
			return reconcile.Result{}, err
		}

		scanningPodName, err := scanning.LaunchScanningPod(r.client, r.config, fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), string(instance.ObjectMeta.UID), imageScanningRequestMetadataKey)

		if err != nil {
			errorMessage := fmt.Sprintf("Error Occurred Creating Scanning Pod '%v'", err)

			logrus.Errorf(errorMessage)

			err = scanning.UpdateOnImageScanningInitializationFailure(r.client, errorMessage, *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		}

		logrus.Infof("Scanning Pod Launched '%s'", scanningPodName)

		err = scanning.UpdateOnScanningPodLaunch(r.client, fmt.Sprintf("Scanning Pod Launched '%s'", scanningPodName), dockerImageID, *instance)

		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil

		// Not an empty phase so check the status of the runnning scanning pods
	} else {

		pod := &corev1.Pod{}
		err := r.client.Get(context.TODO(), request.NamespacedName, instance)
		podOwnerAnnotation := pod.Annotations[common.CopOwnerAnnotation]
		podMetadataKey, _ := cache.MetaNamespaceKeyFunc(pod)

		podNamespace, podName, _ := cache.SplitMetaNamespaceKey(podMetadataKey)

		// Defensive mechanisms
		if pod.ObjectMeta.GetAnnotations() == nil || pod.ObjectMeta.GetAnnotations()[common.CopOwnerAnnotation] == "" || pod.ObjectMeta.GetAnnotations()[common.CopTypeAnnotation] == "" {
			return reconcile.Result{}, nil
		}

		// Check if ImageScanningRequest has already been marked as Succeeded or Failed
		if instance.Status.Phase == images.PhaseCompleted || instance.Status.Phase == images.PhaseFailed {
			return reconcile.Result{}, nil
		}

		// Check to verfiy ImageSigningRequest is in phase Running
		if instance.Status.Phase != images.PhaseRunning {
			return reconcile.Result{}, nil
		}

		// Check if Failed
		if pod.Status.Phase == corev1.PodFailed {
			logrus.Infof("Scanning Pod Failed. Updating ImageSiginingRequest %s", podOwnerAnnotation)

			err = scanning.UpdateOnImageScanningCompletionError(r.client, fmt.Sprintf("Scanning Pod Failed '%v'", err), *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil

		} else if pod.Status.Phase == corev1.PodRunning {
			logrus.Infof("Scanning Pod '%s' in Namespace '%s' is Running. Waiting for Scan to Complete...", podName, podNamespace)

			var delay = time.Duration(10) * time.Second
			attempts := 20

			c1 := make(chan bool)

			go func() {

				for i := 1; i <= attempts; i++ {

					healthRequest := r.k8sclient.CoreV1().RESTClient().Get().
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

				openSCAPReportResult := r.k8sclient.CoreV1().RESTClient().Get().
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

					err = scanning.UpdateOnImageScanningCompletionError(r.client, "OpenSCAP Report Retrieval Failure", *instance)

					if err != nil {
						return reconcile.Result{}, err
					}

					return reconcile.Result{}, nil
				}

				err = xml.Unmarshal(resp, &openSCAPReport)

				if err != nil {
					logrus.Errorf("Failed Unmarshalling OpenSCAP Report %v", err)

					err = scanning.UpdateOnImageScanningCompletionError(r.client, "Failed Unmarshalling OpenSCAP Report", *instance)

					if err != nil {
						return reconcile.Result{}, err
					}

					return reconcile.Result{}, nil
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

				err = scanning.UpdateOnImageScanningCompletionSuccess(r.client, "Image Scanned", totalRules, passedRules, failedRules, *instance)

				if err != nil {
					return reconcile.Result{}, err
				}

				// Best Effort Delete Scanning Pod
				err = scanning.DeleteScanningPod(r.client, podName, podNamespace)

				if err != nil {
					logrus.Warnf("Failed to Delete Scanning Pod '%s': %v", podName, err)
				}

			} else {
				logrus.Infof("Scanning Health Check Could Not Be Validated. Updating ImageScanningRequest %s", podOwnerAnnotation)

				err = scanning.UpdateOnImageScanningCompletionError(r.client, "Health Check Validation Error", *instance)

				if err != nil {
					return reconcile.Result{}, err
				}

				// Best Effort Delete Scanning Pod
				err = scanning.DeleteScanningPod(r.client, podName, podNamespace)

				if err != nil {
					logrus.Warnf("Failed to Delete Scanning Pod '%s': %v", podName, err)
				}

				return reconcile.Result{}, nil

			}

		}
	}
	return reconcile.Result{}, nil
}
