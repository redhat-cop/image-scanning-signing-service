package imagesigningrequest

import (
	"context"
	"fmt"
	"time"

	imagesigningrequestsv1alpha1 "github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	"github.com/redhat-cop/image-security/pkg/controller/config"
	"github.com/redhat-cop/image-security/pkg/controller/imagesigningrequest/signing"
	"github.com/redhat-cop/image-security/pkg/controller/util"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	imageset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
)

var log = logf.Log.WithName("controller_imagesigningrequest")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	configuration := config.LoadConfig()
	client, err := imageset.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil
	}
	return &ReconcileImageSigningRequest{client: mgr.GetClient(), scheme: mgr.GetScheme(), config: configuration, imageClient: client}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagesigningrequest-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ImageSigningRequest
	err = c.Watch(&source.Kind{Type: &imagesigningrequestsv1alpha1.ImageSigningRequest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &imagesigningrequestsv1alpha1.ImageSigningRequest{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileImageSigningRequest implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileImageSigningRequest{}

// ReconcileImageSigningRequest reconciles a ImageSigningRequest object
type ReconcileImageSigningRequest struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	config      config.Config
	imageClient *imageset.ImageV1Client
}

// Reconcile reads that state of the cluster for a ImageSigningRequest object and makes changes based on the state read
// and what is in the ImageSigningRequest.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileImageSigningRequest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ImageSigningRequest")

	// Fetch the ImageSigningRequest instance
	instance := &imagesigningrequestsv1alpha1.ImageSigningRequest{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	imageSigningRequestMetadataKey, _ := cache.MetaNamespaceKeyFunc(instance)
	emptyPhase := imagesigningrequestsv1alpha1.ImageSigningRequestStatus{}.Phase
	if instance.Status.Phase == emptyPhase {
		_, requestIsTag := util.ParseImageStreamTag(instance.Spec.ImageStreamTag)

		//requestImageStreamTag := &imagev1.ImageStreamTag{}
		//err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.ImageStreamTag, Namespace: instance.ObjectMeta.Namespace}, requestImageStreamTag)
		requestImageStreamTag, err := r.imageClient.ImageStreamTags(instance.ObjectMeta.Namespace).Get(instance.Spec.ImageStreamTag, metav1.GetOptions{})
		instance.Status.EndTime = time.Time{}.String() // Need an initial value since time is not nullable
		instance.Status.StartTime = time.Time{}.String()

		if err != nil {

			errorMessage := ""

			if k8serrors.IsNotFound(err) {
				errorMessage = fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", instance.Spec.ImageStreamTag, instance.Namespace)
			} else {
				errorMessage = fmt.Sprintf("Error retrieving ImageStreamTag: %v", err)
			}

			logrus.Warnf(errorMessage)
			err = signing.UpdateOnImageSigningInitializationFailure(r.client, errorMessage, *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil

		}

		dockerImageRegistry, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

		if err != nil {
			return reconcile.Result{}, err
		}
		// TODO - Need to validate this for OCP 4.x
		if requestImageStreamTag.Image.Signatures != nil {
			errorMessage := fmt.Sprintf("Signatures Exist on Image '%s'", dockerImageID)

			logrus.Warnf(errorMessage)

			err = signing.UpdateOnImageSigningInitializationFailure(r.client, errorMessage, *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil

		} else {
			logrus.Infof("No Signatures Exist on Image '%s'", dockerImageID)

			// Setup default values
			gpgSecretName := r.config.GpgSecret
			gpgSignBy := r.config.GpgSignBy

			// Check if Secret if found
			if instance.Spec.SigningKeySecretName != "" {

				signingKeySecret := &corev1.Secret{}
				err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.SigningKeySecretName, Namespace: instance.Namespace}, signingKeySecret)

				if k8serrors.IsNotFound(err) {

					errorMessage := fmt.Sprintf("GPG Secret '%s' Not Found in Namespace '%s'", instance.Spec.SigningKeySecretName, instance.Namespace)
					logrus.Warnf(errorMessage)
					err = signing.UpdateOnImageSigningInitializationFailure(r.client, errorMessage, *instance)

					if err != nil {
						return reconcile.Result{}, err
					}

					return reconcile.Result{}, nil
				}

				logrus.Infof("Copying Secret '%s' to Project '%s'", instance.Spec.SigningKeySecretName, r.config.TargetProject)
				// Create a copy
				signingKeySecretCopy := signingKeySecret.DeepCopy()
				signingKeySecretCopy.Name = string(instance.ObjectMeta.UID)
				signingKeySecretCopy.Namespace = r.config.TargetProject
				signingKeySecretCopy.ResourceVersion = ""
				signingKeySecretCopy.UID = ""

				err = r.client.Create(context.TODO(), signingKeySecretCopy)

				if k8serrors.IsAlreadyExists(err) {
					logrus.Info("Secret already exists. Updating...")
					err = r.client.Update(context.TODO(), signingKeySecretCopy)

				}

				gpgSecretName = signingKeySecretCopy.Name

				if instance.Spec.SigningKeySignBy != "" {
					gpgSignBy = instance.Spec.SigningKeySignBy
				}

			}
			signingPodName, err := signing.LaunchSigningPod(r.client, r.scheme, r.config, instance, fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), dockerImageID, string(instance.ObjectMeta.UID), imageSigningRequestMetadataKey, gpgSecretName, gpgSignBy)

			if err != nil {
				errorMessage := fmt.Sprintf("Error Occurred Creating Signing Pod '%v'", err)

				logrus.Errorf(errorMessage)

				err = signing.UpdateOnImageSigningInitializationFailure(r.client, errorMessage, *instance)

				if err != nil {
					return reconcile.Result{}, err
				}

				return reconcile.Result{}, nil
			}

			logrus.Infof("Signing Pod Launched '%s'", signingPodName)

			err = signing.UpdateOnSigningPodLaunch(r.client, fmt.Sprintf("Signing Pod Launched '%s'", signingPodName), dockerImageID, *instance)

			if err != nil {
				return reconcile.Result{}, err
			}

		}
	}

	return reconcile.Result{}, nil
}
