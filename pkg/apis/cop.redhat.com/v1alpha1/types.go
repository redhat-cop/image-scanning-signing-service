package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageSigningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ImageSigningRequestSpec   `json:"spec"`
	Status            ImageSigningRequestStatus `json:"status,omitempty"`
}

type ImageSigningRequestSpec struct {
	ImageStreamTag string `json:"imageStreamTag"`
	SecretName     string `json:"secretName,omitempty"`
}

type ImageSigningRequestStatus struct {
	State         string `json:"state,omitempty"`
	Message       string `json:"message,omitempty"`
	Phase         Phase  `json:"phase,omitempty"`
	SignedImage   string `json:"signedImage,omitempty"`
	UnsignedImage string `json:"unsignedImage,omitempty"`
}

type Phase string

const (
	PhaseRunning   Phase = "Running"
	PhaseCompleted Phase = "Completed"
	PhaseFailed    Phase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageSigningRequestList is a list of ImageSigningRequest resources
type ImageSigningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImageSigningRequest `json:"items"`
}
