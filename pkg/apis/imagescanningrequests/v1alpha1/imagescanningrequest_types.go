package v1alpha1

import (
	images "github.com/redhat-cop/image-security/pkg/controller/images"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ImageScanningRequestSpec struct {
	ImageStreamTag string `json:"imageStreamTag"`
}
type ImageScanningRequestStatus struct {
	Conditions []images.ImageExecutionCondition `json:"conditions,omitempty"`
	Phase      images.ImageExecutionPhase       `json:"phase,omitempty"`
	ScanResult ScanResult                       `json:"scanResult,omitempty"`
	StartTime  metav1.Time                      `json:"startTime,omitempty"`
	EndTime    metav1.Time                      `json:"endTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageScanningRequest is the Schema for the imagescanningrequests API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=imagescanningrequests,scope=Namespaced
type ImageScanningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ImageScanningRequestSpec   `json:"spec"`
	Status            ImageScanningRequestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageScanningRequestList contains a list of ImageScanningRequest
type ImageScanningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageScanningRequest `json:"items"`
}

type ScanResult struct {
	TotalRules  int `json:"totalRules"`
	PassedRules int `json:"passedRules"`
	FailedRules int `json:"failedRules"`
}

func init() {
	SchemeBuilder.Register(&ImageScanningRequest{}, &ImageScanningRequestList{})
}
