package main // import "github.com/redhat-cop/image-scanning-signing-service"

import (
	"time"

	"github.com/golang/glog"
	imageclientset "github.com/openshift/client-go/image/clientset/versioned"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	kubeinformers "k8s.io/client-go/informers"

	///metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/config"
	"github.com/redhat-cop/image-scanning-signing-service/pkg/signals"
	"k8s.io/client-go/kubernetes"

	clientset "github.com/redhat-cop/image-scanning-signing-service/pkg/client/clientset/versioned"
	informers "github.com/redhat-cop/image-scanning-signing-service/pkg/client/informers/externalversions"
)

func main() {

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// config, err := getConfig(*kubeconfig)

	configuration, err := config.LoadConfig()

	if err != nil {
		glog.Errorf("Failed to Create Configuration: %v", err)
		return
	}
	// build the Kubernetes client
	client, err := kubernetes.NewForConfig(&configuration.K8SConfig)
	if err != nil {
		glog.Errorf("Failed to create kubernetes client: %v", err)
		return
	}

	copClient, err := clientset.NewForConfig(&configuration.K8SConfig)
	if err != nil {
		glog.Fatalf("Error building redhatcop clientset: %v", err)
	}

	imageClient, err := imageclientset.NewForConfig(&configuration.K8SConfig)
	if err != nil {
		glog.Fatalf("Error building image clientset: %v", err)
	}

	templateClient, err := templateclientset.NewForConfig(&configuration.K8SConfig)
	if err != nil {
		glog.Fatalf("Error building template clientset: %v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, time.Second*30)
	copInformerFactory := informers.NewSharedInformerFactory(copClient, time.Second*30)

	controller := NewController(client, copClient, imageClient, templateClient, kubeInformerFactory, copInformerFactory, configuration)

	go kubeInformerFactory.Start(stopCh)
	go copInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}

	// list pods
	/*
		pods, err := client.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			glog.Errorf("Failed to retrieve pods: %v", err)
			return
		}
		for _, p := range pods.Items {
			glog.V(3).Infof("Found pods: %s/%s", p.Namespace, p.Name)
		}
	*/

	/*
		list, err := copClient.CopV1alpha1().ImageSigningRequests("").List(metav1.ListOptions{})
		if err != nil {
			glog.Fatalf("Error listing all imagesigningrequests: %v", err)
		}

		for _, isr := range list.Items {
			glog.V(3).Infof("Found ImageSigningRequest: %s/%s", isr.Namespace, isr.Name)
		}
	*/

}
