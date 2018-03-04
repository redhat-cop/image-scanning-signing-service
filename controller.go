package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	//	"github.com/openshift/origin/pkg/apps/generated/clientset"
	imagev1 "github.com/openshift/api/image/v1"
	imageclientset "github.com/openshift/client-go/image/clientset/versioned"
	imagescheme "github.com/openshift/client-go/image/clientset/versioned/scheme"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	copv1 "github.com/redhat-cop/image-scanning-signing-service/pkg/apis/cop.redhat.com/v1alpha1"
	copclientset "github.com/redhat-cop/image-scanning-signing-service/pkg/client/clientset/versioned"
	copscheme "github.com/redhat-cop/image-scanning-signing-service/pkg/client/clientset/versioned/scheme"
	informers "github.com/redhat-cop/image-scanning-signing-service/pkg/client/informers/externalversions"
	listers "github.com/redhat-cop/image-scanning-signing-service/pkg/client/listers/cop/v1alpha1"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/config"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	controllerAgentName = "image-scan-sign-controller"
	ownerAnnotation     = "cop.redhat.com/owner"
)

type Controller struct {
	podsLister                    corelisters.PodLister
	podsSynced                    cache.InformerSynced
	imagesigningrequestLister     listers.ImageSigningRequestLister
	imagesigningrequestSynced     cache.InformerSynced
	kubeclientset                 kubernetes.Interface
	copclientset                  copclientset.Interface
	imageclientset                imageclientset.Interface
	templateclientset             templateclientset.Interface
	workqueueImageSigningRequests workqueue.RateLimitingInterface
	workqueuePods                 workqueue.RateLimitingInterface
	recorder                      record.EventRecorder
	configuration                 config.Config
}

func NewController(
	kubeclientset kubernetes.Interface,
	copclientset copclientset.Interface,
	imageclientset imageclientset.Interface,
	templateclientset templateclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	copInformerFactory informers.SharedInformerFactory,
	configuration config.Config) *Controller {

	// obtain references to shared index informers for the Deployment and Foo
	// types.
	podInformer := kubeInformerFactory.Core().V1().Pods()
	imageSignImageScanInformer := copInformerFactory.Cop().V1alpha1().ImageSigningRequests()

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	copscheme.AddToScheme(scheme.Scheme)
	imagescheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:                 kubeclientset,
		copclientset:                  copclientset,
		imageclientset:                imageclientset,
		templateclientset:             templateclientset,
		podsLister:                    podInformer.Lister(),
		podsSynced:                    podInformer.Informer().HasSynced,
		imagesigningrequestLister:     imageSignImageScanInformer.Lister(),
		imagesigningrequestSynced:     imageSignImageScanInformer.Informer().HasSynced,
		workqueueImageSigningRequests: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ImageSigningRequests"),
		workqueuePods:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ImageSigningPods"),
		recorder:                      recorder,
		configuration:                 configuration,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when ImageSignScanRequest resources change
	imageSignImageScanInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueImageSignScanRequest,
	})

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Foo resource will enqueue that Foo resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*corev1.Pod)
			oldPod := old.(*corev1.Pod)
			if newPod.ResourceVersion == oldPod.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueueImageSigningRequests.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced, c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorkerImageSigningRequests, time.Second, stopCh)
		go wait.Until(c.runWorkerPods, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueImageSignScanRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueueImageSigningRequests.AddRateLimited(key)
}

func (c *Controller) enqueuePods(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueuePods.AddRateLimited(key)
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorkerImageSigningRequests() {
	for c.processNextImageSigningRequestWorkItem() {
	}
}

func (c *Controller) runWorkerPods() {
	for c.processNextPodsWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextImageSigningRequestWorkItem() bool {
	obj, shutdown := c.workqueueImageSigningRequests.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {

		defer c.workqueueImageSigningRequests.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {

			c.workqueueImageSigningRequests.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in ImageSigningRequest workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandlerImageSigningRequests, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandlerImageSigningRequests(key); err != nil {
			return fmt.Errorf("Error syncing ImageSigningRequest '%s': %s", key, err.Error())
		}

		c.workqueueImageSigningRequests.Forget(obj)
		glog.Infof("Successfully synced ImageSigningRequest '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextPodsWorkItem() bool {
	obj, shutdown := c.workqueuePods.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {

		defer c.workqueuePods.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {

			c.workqueuePods.Forget(obj)
			runtime.HandleError(fmt.Errorf("Expected string in Pods workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandlerPods(key); err != nil {
			return fmt.Errorf("Error syncing Pods '%s': %s", key, err.Error())
		}

		c.workqueuePods.Forget(obj)
		glog.Infof("Successfully synced Pods '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandlerImageSigningRequests(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	imageSigningRequest, err := c.imagesigningrequestLister.ImageSigningRequests(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if k8serrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("ImageSigningRequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Trigger new Signing Template action if no status found
	emptyPhase := copv1.ImageSigningRequestStatus{}.Phase
	if imageSigningRequest.Status.Phase == emptyPhase {

		_, requestIsTag := parseImageStreamTag(imageSigningRequest.Spec.ImageStreamTag)

		requestImageStreamTag, err := c.imageclientset.ImageV1().ImageStreamTags(imageSigningRequest.Namespace).Get(imageSigningRequest.Spec.ImageStreamTag, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			glog.Warningf("Cannot Find ImageStreamTag %s in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)

			err = c.updateOnInitializationFailure(fmt.Sprintf("ImageStreamTag %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace), *imageSigningRequest)

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
			glog.Warningf("Signatures Exist on Image '%s'", dockerImageID)

			err = c.updateOnInitializationFailure(fmt.Sprintf("Signatures Exist on Image '%s'", dockerImageID), *imageSigningRequest)

			if err != nil {
				return err
			}

			return nil

		} else {
			glog.Infof("No Signatures Exist on Image '%s'", dockerImageID)

			signingPodName, err := c.launchPod(fmt.Sprintf("%s:%s", dockerImageRegistry, requestIsTag), dockerImageID, string(imageSigningRequest.ObjectMeta.UID), key)

			if err != nil {
				glog.Errorf("Error Occurred Creating Signing Pod '%v'", err)

				err = c.updateOnInitializationFailure(fmt.Sprintf("Error Occurred Creating Signing Pod '%v'", err), *imageSigningRequest)

				if err != nil {
					return err
				}

				return nil
			}

			glog.Infof("Signing Pod Launched '%s'", signingPodName)

			err = c.updateOnSigningPodLaunch(fmt.Sprintf("Signing Pod Launched '%s'", signingPodName), dockerImageID, *imageSigningRequest)

			if err != nil {
				return err
			}

			return nil

		}

	}

	return nil
}

func (c *Controller) syncHandlerPods(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	glog.Infof("Syncing Pod %s in namespace %s", name, namespace)

	pod, err := c.kubeclientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("pod '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	podOwnerAnnotation := pod.Annotations[ownerAnnotation]

	isrNamespace, isrName, err := cache.SplitMetaNamespaceKey(podOwnerAnnotation)

	imageSigningRequest, err := c.imagesigningrequestLister.ImageSigningRequests(isrNamespace).Get(isrName)

	if err != nil {
		glog.Warningf("Could not find ImageSigningRequest '%s' from pod '%s'", podOwnerAnnotation, key)
		return err
	}

	// Check to verfiy ImageSigningRequest is in phase Running
	if imageSigningRequest.Status.Phase != copv1.PhaseRunning {
		glog.V(4).Info("ImageSigingRequest '%s' is Not Currently Running", podOwnerAnnotation)
		return nil
	}

	// Check if Failed
	if pod.Status.Phase == corev1.PodFailed {
		glog.Infof("Signing Pod Failed. Updating ImageSiginingRequest %s", podOwnerAnnotation)

		err = c.updateOnCompletionError(fmt.Sprintf("Signing Pod Failed '%v'", err), *imageSigningRequest)

		if err != nil {
			return err
		}

		return nil

	} else if pod.Status.Phase == corev1.PodSucceeded {

		requestImageStreamTag, err := c.imageclientset.ImageV1().ImageStreamTags(imageSigningRequest.Namespace).Get(imageSigningRequest.Spec.ImageStreamTag, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			glog.Warningf("Cannot Find ImageStreamTag %s in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace)

			err = c.updateOnCompletionError(fmt.Sprintf("ImageStream %s Not Found in Namespace %s", imageSigningRequest.Spec.ImageStreamTag, imageSigningRequest.Namespace), *imageSigningRequest)

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
			glog.Infof("Signing Pod Succeeded. Updating ImageSiginingRequest %s", pod.Annotations[ownerAnnotation])

			err = c.updateOnCompletionSuccess("Image Signed", dockerImageID, *imageSigningRequest)

			if err != nil {
				return err
			}

		} else {
			err = c.updateOnCompletionError(fmt.Sprintf("No Signature Exists on Image '%s' After Signing Completed", dockerImageID), *imageSigningRequest)

			if err != nil {
				return err
			}

		}

		return nil
	}

	return nil
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	if objectAnnotations := object.GetAnnotations(); objectAnnotations != nil {
		if objectAnnotations != nil && objectAnnotations[ownerAnnotation] != "" {

			pod := object.(*corev1.Pod)

			namespace, name, err := cache.SplitMetaNamespaceKey(objectAnnotations[ownerAnnotation])

			// Find associated ImageSigningRequest
			_, err = c.imagesigningrequestLister.ImageSigningRequests(namespace).Get(name)
			if err != nil {
				glog.Infof("ignoring orphaned object '%s' of ImageSigningRequest '%s'", object.GetSelfLink(), name)
				return
			}

			// Queue the pod if it has failed or succeeded
			switch pod.Status.Phase {
			case corev1.PodSucceeded, corev1.PodFailed:
				glog.Info("Enquing Pod as it has succeeded or failed")
				c.enqueuePods(pod)
			default:
				glog.V(4).Infof("Ignoring Pod in phase %s", pod.Status.Phase)
			}

		}
	}
}

func (c *Controller) launchPod(image string, imageDigest string, ownerID string, ownerReference string) (string, error) {

	pod, err := util.CreateSigningPod(c.configuration.SignScanImage, image, imageDigest, ownerID, ownerReference, c.configuration.TargetServiceAccount, c.configuration.GpgSecret, c.configuration.GpgSignBy)

	if err != nil {
		glog.Errorf("Error Generating Pod: %v'", err)
		return "", err
	}

	createdPod, err := c.kubeclientset.CoreV1().Pods(c.configuration.TargetProject).Create(pod)

	if err != nil {
		glog.Errorf("Error Creating Pod: %v'", err)
		return "", err
	}

	var key string
	if key, err = cache.MetaNamespaceKeyFunc(createdPod); err != nil {
		runtime.HandleError(err)
		return "", err
	}

	return key, nil
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

func (c *Controller) updateImageSigningRequest(imageSigningRequest *copv1.ImageSigningRequest, condition copv1.ImageSigningCondition, phase copv1.ImageSigningPhase) error {

	imageSigningRequest.Status.Conditions = append(imageSigningRequest.Status.Conditions, condition)
	imageSigningRequest.Status.Phase = phase

	_, err := c.copclientset.CopV1alpha1().ImageSigningRequests(imageSigningRequest.Namespace).Update(imageSigningRequest)

	return err
}

func newImageSigningCondition(message string, conditionStatus corev1.ConditionStatus, conditionType copv1.ImageSigningConditionType) copv1.ImageSigningCondition {

	return copv1.ImageSigningCondition{
		LastTransitionTime: metav1.Now(),
		Message:            message,
		Status:             conditionStatus,
		Type:               conditionType,
	}

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

func (c *Controller) updateOnInitializationFailure(message string, imageSigningRequest copv1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionFalse, copv1.ImageSigningConditionInitialization)

	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return c.updateImageSigningRequest(imageSigningRequestCopy, condition, copv1.PhaseFailed)
}

func (c *Controller) updateOnSigningPodLaunch(message string, unsignedImage string, imageSigningRequest copv1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionTrue, copv1.ImageSigningConditionInitialization)

	imageSigningRequestCopy.Status.UnsignedImage = unsignedImage
	imageSigningRequestCopy.Status.StartTime = condition.LastTransitionTime

	return c.updateImageSigningRequest(imageSigningRequestCopy, condition, copv1.PhaseRunning)
}

func (c *Controller) updateOnCompletionError(message string, imageSigningRequest copv1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionFalse, copv1.ImageSigningConditionFinished)

	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return c.updateImageSigningRequest(imageSigningRequestCopy, condition, copv1.PhaseFailed)
}

func (c *Controller) updateOnCompletionSuccess(message string, signedImage string, imageSigningRequest copv1.ImageSigningRequest) error {
	imageSigningRequestCopy := imageSigningRequest.DeepCopy()

	condition := newImageSigningCondition(message, corev1.ConditionFalse, copv1.ImageSigningConditionFinished)

	imageSigningRequestCopy.Status.SignedImage = signedImage
	imageSigningRequestCopy.Status.EndTime = condition.LastTransitionTime

	return c.updateImageSigningRequest(imageSigningRequestCopy, condition, copv1.PhaseCompleted)
}
