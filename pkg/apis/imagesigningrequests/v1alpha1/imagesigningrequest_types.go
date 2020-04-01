package v1alpha1

import (
	images "github.com/redhat-cop/image-security/pkg/controller/images"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSigningRequestSpec defines the desired state of ImageSigningRequest
// +k8s:openapi-gen=true
type ImageSigningRequestSpec struct {
	ImageStreamTag       string `json:"imageStreamTag"`
	SigningKeySecretName string `json:"signingKeySecretName,omitempty"`
	SigningKeySignBy     string `json:"signingKeySignBy,omitempty"`
}

// ImageSigningRequestStatus defines the observed state of ImageSigningRequest
// +k8s:openapi-gen=true
type ImageSigningRequestStatus struct {
	Conditions    []images.ImageExecutionCondition `json:"conditions,omitempty"`
	Phase         images.ImageExecutionPhase       `json:"phase,omitempty"`
	SignedImage   string                           `json:"signedImage,omitempty"`
	UnsignedImage string                           `json:"unsignedImage,omitempty"`
	StartTime     string                           `json:"startTime,omitempty"`
	EndTime       string                           `json:"endTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageSigningRequest is the Schema for the imagesigningrequests API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=imagesigningrequests,scope=Namespaced
type ImageSigningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSigningRequestSpec   `json:"spec,omitempty"`
	Status ImageSigningRequestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageSigningRequestList contains a list of ImageSigningRequest
type ImageSigningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageSigningRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageSigningRequest{}, &ImageSigningRequestList{})
}
