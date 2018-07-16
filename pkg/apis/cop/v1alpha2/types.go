package v1alpha2

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageSigningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImageSigningRequest `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageScanningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImageSigningRequest `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageSigningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ImageSigningRequestSpec   `json:"spec"`
	Status            ImageSigningRequestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageScanningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ImageScanningRequestSpec   `json:"spec"`
	Status            ImageScanningRequestStatus `json:"status,omitempty"`
}

type ImageSigningRequestSpec struct {
	ImageStreamTag       string `json:"imageStreamTag"`
	SigningKeySecretName string `json:"signingKeySecretName,omitempty"`
	SigningKeySignBy     string `json:"signingKeySignBy,omitempty"`
}
type ImageSigningRequestStatus struct {
	Conditions    []ImageExecutionCondition `json:"conditions,omitempty"`
	Phase         ImageExecutionPhase       `json:"phase,omitempty"`
	SignedImage   string                    `json:"signedImage,omitempty"`
	UnsignedImage string                    `json:"unsignedImage,omitempty"`
	StartTime     metav1.Time               `json:"startTime,omitempty"`
	EndTime       metav1.Time               `json:"endTime,omitempty"`
}

type ImageScanningRequestSpec struct {
	ImageStreamTag string `json:"imageStreamTag"`
}
type ImageScanningRequestStatus struct {
	Conditions []ImageExecutionCondition `json:"conditions,omitempty"`
	Phase      ImageExecutionPhase       `json:"phase,omitempty"`
	ScanResult ScanResult                `json:"scanResult,omitempty"`
	StartTime  metav1.Time               `json:"startTime,omitempty"`
	EndTime    metav1.Time               `json:"endTime,omitempty"`
}

type ScanResult struct {
	TotalRules  int `json:"totalRules"`
	PassedRules int `json:"passedRules"`
	FailedRules int `json:"failedRules"`
}

type ImageExecutionCondition struct {
	Status             v1.ConditionStatus          `json:"status,omitempty"`
	Message            string                      `json:"message,omitempty"`
	LastTransitionTime metav1.Time                 `json:"lastTransitionTime,omitempty"`
	Type               ImageExecutionConditionType `json:"type,omitempty"`
}

type ImageExecutionPhase string

const (
	PhaseRunning   ImageExecutionPhase = "Running"
	PhaseCompleted ImageExecutionPhase = "Completed"
	PhaseFailed    ImageExecutionPhase = "Failed"
)

type ImageExecutionConditionType string

const (
	ImageExecutionConditionInitialization = "Initialization"
	ImageExecutionConditionSigning        = "Signing"
	ImageExecutionConditionFinished       = "Finished"
)
