/*
Copyright 2018 The Red Hat Container and PaaS Community of Practice.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1alpha1 "github.com/redhat-cop/image-scanning-signing-service/pkg/apis/cop.redhat.com/v1alpha1"
	scheme "github.com/redhat-cop/image-scanning-signing-service/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ImageSigningRequestsGetter has a method to return a ImageSigningRequestInterface.
// A group's client should implement this interface.
type ImageSigningRequestsGetter interface {
	ImageSigningRequests(namespace string) ImageSigningRequestInterface
}

// ImageSigningRequestInterface has methods to work with ImageSigningRequest resources.
type ImageSigningRequestInterface interface {
	Create(*v1alpha1.ImageSigningRequest) (*v1alpha1.ImageSigningRequest, error)
	Update(*v1alpha1.ImageSigningRequest) (*v1alpha1.ImageSigningRequest, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ImageSigningRequest, error)
	List(opts v1.ListOptions) (*v1alpha1.ImageSigningRequestList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ImageSigningRequest, err error)
	ImageSigningRequestExpansion
}

// imageSigningRequests implements ImageSigningRequestInterface
type imageSigningRequests struct {
	client rest.Interface
	ns     string
}

// newImageSigningRequests returns a ImageSigningRequests
func newImageSigningRequests(c *CopV1alpha1Client, namespace string) *imageSigningRequests {
	return &imageSigningRequests{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the imageSigningRequest, and returns the corresponding imageSigningRequest object, and an error if there is any.
func (c *imageSigningRequests) Get(name string, options v1.GetOptions) (result *v1alpha1.ImageSigningRequest, err error) {
	result = &v1alpha1.ImageSigningRequest{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ImageSigningRequests that match those selectors.
func (c *imageSigningRequests) List(opts v1.ListOptions) (result *v1alpha1.ImageSigningRequestList, err error) {
	result = &v1alpha1.ImageSigningRequestList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested imageSigningRequests.
func (c *imageSigningRequests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a imageSigningRequest and creates it.  Returns the server's representation of the imageSigningRequest, and an error, if there is any.
func (c *imageSigningRequests) Create(imageSigningRequest *v1alpha1.ImageSigningRequest) (result *v1alpha1.ImageSigningRequest, err error) {
	result = &v1alpha1.ImageSigningRequest{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		Body(imageSigningRequest).
		Do().
		Into(result)
	return
}

// Update takes the representation of a imageSigningRequest and updates it. Returns the server's representation of the imageSigningRequest, and an error, if there is any.
func (c *imageSigningRequests) Update(imageSigningRequest *v1alpha1.ImageSigningRequest) (result *v1alpha1.ImageSigningRequest, err error) {
	result = &v1alpha1.ImageSigningRequest{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		Name(imageSigningRequest.Name).
		Body(imageSigningRequest).
		Do().
		Into(result)
	return
}

// Delete takes name of the imageSigningRequest and deletes it. Returns an error if one occurs.
func (c *imageSigningRequests) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *imageSigningRequests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("imagesigningrequests").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched imageSigningRequest.
func (c *imageSigningRequests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ImageSigningRequest, err error) {
	result = &v1alpha1.ImageSigningRequest{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("imagesigningrequests").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
