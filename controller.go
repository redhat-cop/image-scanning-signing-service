package main

import (
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
	"k8s.io/apimachinery/pkg/api/errors"
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
			return fmt.Errorf("error syncing ImageSigningRequest '%s': %s", key, err.Error())
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
			runtime.HandleError(fmt.Errorf("expected string in Pods workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandlerPods(key); err != nil {
			return fmt.Errorf("error syncing Pods '%s': %s", key, err.Error())
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
	imagesigningrequest, err := c.imagesigningrequestLister.ImageSigningRequests(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("imagesigningrequest '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Trigger new Signing Template action if no status found
	emptyStatus := copv1.ImageSigningRequestStatus{}
	emptyPhase := copv1.ImageSigningRequestStatus{}.Phase
	if imagesigningrequest.Status == emptyStatus && imagesigningrequest.Status.Phase == emptyPhase {

		// TODO: Consolidate this logic with similar logic during confirmation phase
		requestIsName, requestIsTag := parseImageStreamTag(imagesigningrequest.Spec.ImageStreamTag)

		requestImageStream, err := c.imageclientset.ImageV1().ImageStreams(imagesigningrequest.Namespace).Get(requestIsName, metav1.GetOptions{})

		if errors.IsNotFound(err) {
			glog.Warningf("Cannot Find ImageStream %s in Namespace %s", requestIsName, imagesigningrequest.Namespace)

			err = c.updateImageSigningRequest(fmt.Sprintf("ImageStream %s Not Found in Namespace %s", requestIsName, imagesigningrequest.Namespace), "", "", *imagesigningrequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil

		}

		requestImageStreamTagEvent := latestTaggedImage(requestImageStream, requestIsTag)

		if requestImageStreamTagEvent == nil {
			glog.Errorf("Unable to locate tag '%s' on ImageStream '%s'", requestIsTag, requestIsName)

			err = c.updateImageSigningRequest(fmt.Sprintf("Unable to locate tag '%s' on ImageStream '%s'", requestIsTag, requestIsName), "", "", *imagesigningrequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil
		}

		image, err := c.imageclientset.ImageV1().Images().Get(requestImageStreamTagEvent.Image, metav1.GetOptions{})

		if err != nil {
			runtime.HandleError(fmt.Errorf("Invalid image: %s", requestImageStreamTagEvent.Image))

			err = c.updateImageSigningRequest(fmt.Sprintf("Invalid image '%s'", requestImageStreamTagEvent.Image), "", "", *imagesigningrequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil
		}

		if image.Signatures != nil {
			glog.Warningf("Signatures Exist on Image '%s'", requestImageStreamTagEvent.Image)

			err = c.updateImageSigningRequest(fmt.Sprintf("Signatures Exist on Image '%s'", requestImageStreamTagEvent.Image), "", requestImageStreamTagEvent.Image, *imagesigningrequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil

		} else {
			glog.Infof("No Signatures Exist on Image '%s'", requestImageStreamTagEvent.Image)

			dockerImageReferenceContent := strings.Split(requestImageStreamTagEvent.DockerImageReference, "@")

			signingPodName, err := c.launchPod(fmt.Sprintf("%s:%s", dockerImageReferenceContent[0], requestIsTag), dockerImageReferenceContent[1], string(imagesigningrequest.ObjectMeta.UID), key)

			if err != nil {
				glog.Errorf("Error Occurred Creating Signing Pod '%v'", err)

				err = c.updateImageSigningRequest(fmt.Sprintf("Error Occurred Creating Signing Pod '%v'", err), requestImageStreamTagEvent.Image, "", *imagesigningrequest, copv1.PhaseFailed)

				if err != nil {
					return err
				}

				return nil
			}

			glog.Infof("Signing Pod Launched '%s'", signingPodName)

			err = c.updateImageSigningRequest(fmt.Sprintf("Signing Pod Launched '%s'", signingPodName), requestImageStreamTagEvent.Image, "", *imagesigningrequest, copv1.PhaseRunning)

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
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("pod '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	podOwnerAnnotation := pod.Annotations[ownerAnnotation]

	isrNamespace, isrName, err := cache.SplitMetaNamespaceKey(podOwnerAnnotation)

	imageSigningRequest, err := c.imagesigningrequestLister.ImageSigningRequests(isrNamespace).Get(isrName)

	if err != nil {
		glog.Warningf("could not find imagesigningrequest '%s' from pod '%s'", podOwnerAnnotation, key)
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

		err = c.updateImageSigningRequest(fmt.Sprintf("Signing Pod Failed '%v'", err), "", "", *imageSigningRequest, copv1.PhaseFailed)

		if err != nil {
			return err
		}

		return nil

	} else if pod.Status.Phase == corev1.PodSucceeded {

		// TODO: Need to check if latestimage has a signature

		// TODO: Consolidate this logic with similar logic during confirmation phase
		requestIsName, requestIsTag := parseImageStreamTag(imageSigningRequest.Spec.ImageStreamTag)

		requestImageStream, err := c.imageclientset.ImageV1().ImageStreams(imageSigningRequest.Namespace).Get(requestIsName, metav1.GetOptions{})

		if errors.IsNotFound(err) {
			glog.Warningf("Cannot Find ImageStream %s in Namespace %s", requestIsName, imageSigningRequest.Namespace)

			err = c.updateImageSigningRequest(fmt.Sprintf("ImageStream %s Not Found in Namespace %s", requestIsName, imageSigningRequest.Namespace), "", "", *imageSigningRequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil

		}

		requestImageStreamTagEvent := latestTaggedImage(requestImageStream, requestIsTag)

		if requestImageStreamTagEvent == nil {
			glog.Errorf("Unable to locate tag '%s' on ImageStream '%s'", requestIsTag, requestIsName)

			err = c.updateImageSigningRequest(fmt.Sprintf("Unable to locate tag '%s' on ImageStream '%s'", requestIsTag, requestIsName), "", "", *imageSigningRequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil
		}

		image, err := c.imageclientset.ImageV1().Images().Get(requestImageStreamTagEvent.Image, metav1.GetOptions{})

		if err != nil {
			runtime.HandleError(fmt.Errorf("invalid image: %s", requestImageStreamTagEvent.Image))

			err = c.updateImageSigningRequest(fmt.Sprintf("invalid image '%s'", requestImageStreamTagEvent.Image), "", "", *imageSigningRequest, copv1.PhaseFailed)

			if err != nil {
				return err
			}

			return nil
		}

		if image.Signatures != nil {
			glog.Infof("Signing Pod Succeeded. Updating ImageSiginingRequest %s", pod.Annotations[ownerAnnotation])

			err = c.updateImageSigningRequest("Image Signed", "", requestImageStreamTagEvent.Image, *imageSigningRequest, copv1.PhaseCompleted)

			if err != nil {
				return err
			}

		} else {
			err = c.updateImageSigningRequest(fmt.Sprintf("No Signature Exists on Image '%s' After Signing Completed", requestImageStreamTagEvent.Image), "", "", *imageSigningRequest, copv1.PhaseFailed)

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

/*
func (c *Controller) launchTemplate(image string, imageDigest string, ownerID string, ownerReference string) (string, error) {

	templateNamespace, templateName, err := cache.SplitMetaNamespaceKey(c.configuration.SigningTemplate)

	// Check that Template Exists
	template, err := c.templateclientset.TemplateV1().Templates(templateNamespace).Get(templateName, metav1.GetOptions{})

	if err != nil {
		glog.Errorf("Failed to Get Template: %v", err)
		return "", err
	}

	glog.Infof("Template Information: %v", template)

	// TODO: Add some logic to do parameter checking

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: ownerID,
		},
		Data: map[string][]byte{
			"OWNER_REFERENCE": []byte(ownerReference),
			"IMAGE_TO_SIGN":   []byte(image),
			"IMAGE_DIGEST":    []byte(imageDigest),
			"NAMESPACE":       []byte(c.configuration.TargetProject),
			"SIGN_BY":         []byte(c.configuration.GpgSignBy),
		},
	}

	_, err = c.kubeclientset.CoreV1().Secrets(c.configuration.TargetProject).Create(secret)

	if err != nil {
		glog.Errorf("Failed to Create Secret: %v", err)
		return "", err
	}

	templateinstance := &templateapi.TemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: ownerID,
		},
		Spec: templateapi.TemplateInstanceSpec{
			Template: *template,
			Secret: &corev1.LocalObjectReference{
				Name: ownerID,
			},
		},
	}

	_, err = c.templateclientset.Template().TemplateInstances(c.configuration.TargetProject).Create(templateinstance)

	if err != nil {
		glog.Errorf("Failed to create template instance: %v", err)
		return "", err
	}

	// wait for templateinstance controller to do its thing
	err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
		templateinstance, err = c.templateclientset.TemplateV1().TemplateInstances(c.configuration.TargetProject).Get(ownerID, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, c := range templateinstance.Status.Conditions {
			if c.Reason == "Failed" && c.Status == corev1.ConditionTrue {
				return false, fmt.Errorf("failed condition: %s", c.Message)
			}
			if c.Reason == "Created" && c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	})

	if err != nil {
		glog.Errorf("There was an error instantiating the template: %v", err)
		return "", err
	}

	return "", nil
}
*/

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

func (c *Controller) updateImageSigningRequest(message string, unsignedImage string, signedImage string, imageSigningRequest copv1.ImageSigningRequest, phase copv1.Phase) error {

	imagesigningrequestCopy := imageSigningRequest.DeepCopy()
	imagesigningrequestCopy.Status.Message = message
	imagesigningrequestCopy.Status.Phase = phase

	if unsignedImage != "" {
		imagesigningrequestCopy.Status.UnsignedImage = unsignedImage
	}

	if signedImage != "" {
		imagesigningrequestCopy.Status.SignedImage = signedImage
	}

	_, err := c.copclientset.CopV1alpha1().ImageSigningRequests(imageSigningRequest.Namespace).Update(imagesigningrequestCopy)

	return err
}
