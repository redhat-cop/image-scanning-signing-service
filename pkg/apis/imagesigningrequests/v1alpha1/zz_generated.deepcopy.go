// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1alpha1

import (
	images "github.com/redhat-cop/image-security/pkg/controller/images"
	runtime "k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSigningRequest) DeepCopyInto(out *ImageSigningRequest) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSigningRequest.
func (in *ImageSigningRequest) DeepCopy() *ImageSigningRequest {
	if in == nil {
		return nil
	}
	out := new(ImageSigningRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageSigningRequest) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSigningRequestList) DeepCopyInto(out *ImageSigningRequestList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ImageSigningRequest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSigningRequestList.
func (in *ImageSigningRequestList) DeepCopy() *ImageSigningRequestList {
	if in == nil {
		return nil
	}
	out := new(ImageSigningRequestList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageSigningRequestList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSigningRequestSpec) DeepCopyInto(out *ImageSigningRequestSpec) {
	*out = *in
	if in.ContainerImage != nil {
		in, out := &in.ContainerImage, &out.ContainerImage
		*out = new(core.ObjectReference)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSigningRequestSpec.
func (in *ImageSigningRequestSpec) DeepCopy() *ImageSigningRequestSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSigningRequestSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSigningRequestStatus) DeepCopyInto(out *ImageSigningRequestStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]images.ImageExecutionCondition, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSigningRequestStatus.
func (in *ImageSigningRequestStatus) DeepCopy() *ImageSigningRequestStatus {
	if in == nil {
		return nil
	}
	out := new(ImageSigningRequestStatus)
	in.DeepCopyInto(out)
	return out
}
