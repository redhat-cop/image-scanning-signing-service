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

package fake

import (
	v1alpha1 "github.com/redhat-cop/image-scanning-signing-service/pkg/apis/cop.redhat.com/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeImageSigningRequests implements ImageSigningRequestInterface
type FakeImageSigningRequests struct {
	Fake *FakeCopV1alpha1
	ns   string
}

var imagesigningrequestsResource = schema.GroupVersionResource{Group: "cop.redhat.com", Version: "v1alpha1", Resource: "imagesigningrequests"}

var imagesigningrequestsKind = schema.GroupVersionKind{Group: "cop.redhat.com", Version: "v1alpha1", Kind: "ImageSigningRequest"}

// Get takes name of the imageSigningRequest, and returns the corresponding imageSigningRequest object, and an error if there is any.
func (c *FakeImageSigningRequests) Get(name string, options v1.GetOptions) (result *v1alpha1.ImageSigningRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(imagesigningrequestsResource, c.ns, name), &v1alpha1.ImageSigningRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImageSigningRequest), err
}

// List takes label and field selectors, and returns the list of ImageSigningRequests that match those selectors.
func (c *FakeImageSigningRequests) List(opts v1.ListOptions) (result *v1alpha1.ImageSigningRequestList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(imagesigningrequestsResource, imagesigningrequestsKind, c.ns, opts), &v1alpha1.ImageSigningRequestList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ImageSigningRequestList{}
	for _, item := range obj.(*v1alpha1.ImageSigningRequestList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested imageSigningRequests.
func (c *FakeImageSigningRequests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(imagesigningrequestsResource, c.ns, opts))

}

// Create takes the representation of a imageSigningRequest and creates it.  Returns the server's representation of the imageSigningRequest, and an error, if there is any.
func (c *FakeImageSigningRequests) Create(imageSigningRequest *v1alpha1.ImageSigningRequest) (result *v1alpha1.ImageSigningRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(imagesigningrequestsResource, c.ns, imageSigningRequest), &v1alpha1.ImageSigningRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImageSigningRequest), err
}

// Update takes the representation of a imageSigningRequest and updates it. Returns the server's representation of the imageSigningRequest, and an error, if there is any.
func (c *FakeImageSigningRequests) Update(imageSigningRequest *v1alpha1.ImageSigningRequest) (result *v1alpha1.ImageSigningRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(imagesigningrequestsResource, c.ns, imageSigningRequest), &v1alpha1.ImageSigningRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImageSigningRequest), err
}

// Delete takes name of the imageSigningRequest and deletes it. Returns an error if one occurs.
func (c *FakeImageSigningRequests) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(imagesigningrequestsResource, c.ns, name), &v1alpha1.ImageSigningRequest{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeImageSigningRequests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(imagesigningrequestsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ImageSigningRequestList{})
	return err
}

// Patch applies the patch and returns the patched imageSigningRequest.
func (c *FakeImageSigningRequests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ImageSigningRequest, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(imagesigningrequestsResource, c.ns, name, data, subresources...), &v1alpha1.ImageSigningRequest{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImageSigningRequest), err
}
