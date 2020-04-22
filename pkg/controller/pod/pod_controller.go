package pod

import (
	"context"
	"fmt"

	imageset "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	imagesigningrequestsv1alpha1 "github.com/redhat-cop/image-security/pkg/apis/imagesigningrequests/v1alpha1"
	"github.com/redhat-cop/image-security/pkg/controller/common"
	"github.com/redhat-cop/image-security/pkg/controller/images"
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
)

var log = logf.Log.WithName("controller_pod")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Pod Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client, err := imageset.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil
	}
	return &ReconcilePod{client: mgr.GetClient(), scheme: mgr.GetScheme(), imageClient: client}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller

	c, err := controller.New("pod-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Pod
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePod implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePod{}

// ReconcilePod reconciles a Pod object
type ReconcilePod struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	imageClient *imageset.ImageV1Client
}

// Reconcile reads that state of the cluster for a Pod object and makes changes based on the state read
// and what is in the Pod.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePod) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// Fetch the Pod instance
	pod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Defensive mechanisms
	if pod.ObjectMeta.GetAnnotations() == nil || pod.ObjectMeta.GetAnnotations()[common.CopOwnerAnnotation] == "" || pod.ObjectMeta.GetAnnotations()[common.CopTypeAnnotation] == "" {
		return reconcile.Result{}, nil
	}
	// Reduce noise and only log signing pods
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Pod")

	podOwnerAnnotation := pod.Annotations[common.CopOwnerAnnotation]
	podMetadataKey, _ := cache.MetaNamespaceKeyFunc(pod)
	isrNamespace, isrName, err := cache.SplitMetaNamespaceKey(podOwnerAnnotation)

	imageSigningRequest := &imagesigningrequestsv1alpha1.ImageSigningRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageSigningRequest",
			APIVersion: "cop.redhat.com/v1alpha2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      isrName,
			Namespace: isrNamespace,
		},
	}

	err = r.client.Get(context.TODO(), types.NamespacedName{Name: isrName, Namespace: isrNamespace}, imageSigningRequest)
	if err != nil {
		logrus.Warnf("Could not find ImageSigningRequest '%s' from pod '%s'", podOwnerAnnotation, podMetadataKey)
		return reconcile.Result{}, nil
	}

	// Check if ImageSigningRequest has already been marked as Succeeded or Failed
	if imageSigningRequest.Status.Phase == images.PhaseCompleted || imageSigningRequest.Status.Phase == images.PhaseFailed {
		return reconcile.Result{}, nil
	}

	// Check to verfiy ImageSigningRequest is in phase Running
	if imageSigningRequest.Status.Phase != images.PhaseRunning {
		return reconcile.Result{}, nil
	}

	// Check if Failed
	if pod.Status.Phase == corev1.PodFailed {
		logrus.Infof("Signing Pod Failed. Updating ImageSiginingRequest %s", podOwnerAnnotation)

		err = signing.UpdateOnImageSigningCompletionError(r.client, fmt.Sprintf("Signing Pod Failed '%v'", err), *imageSigningRequest)

		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil

	} else if pod.Status.Phase == corev1.PodSucceeded {
		requestImageStreamTag, err := r.imageClient.ImageStreamTags(imageSigningRequest.ObjectMeta.Namespace).Get(imageSigningRequest.Spec.ImageStreamTag, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {

			errorMessage := fmt.Sprintf("ImageStream %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)
			logrus.Warnf(errorMessage)

			err = signing.UpdateOnImageSigningCompletionError(r.client, errorMessage, *imageSigningRequest)

			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil

		}

		_, dockerImageID, err := util.ExtractImageIDFromImageReference(requestImageStreamTag.Image.DockerImageReference)

		if err != nil {
			return reconcile.Result{}, err
		}

		logrus.Infof("Signing Pod Succeeded. Updating ImageSiginingRequest %s", pod.Annotations[common.CopOwnerAnnotation])

		err = signing.UpdateOnImageSigningCompletionSuccess(r.client, "Image Signed", dockerImageID, *imageSigningRequest)

		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
