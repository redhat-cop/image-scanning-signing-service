package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageSigningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ImageSigningRequestSpec   `json:"spec"`
	Status            ImageSigningRequestStatus `json:"status"`
}

type ImageSigningRequestSpec struct {
	ImageStreamTag       string `json:"imageStreamTag"`
	SigningKeySecretName string `json:"signingKeySecretName,omitempty"`
	SigningKeySignBy     string `json:"signingKeySignBy,omitempty"`
}

type ImageSigningRequestStatus struct {
	Conditions    []ImageSigningCondition `json:"conditions,omitempty"`
	Phase         ImageSigningPhase       `json:"phase,omitempty"`
	SignedImage   string                  `json:"signedImage,omitempty"`
	UnsignedImage string                  `json:"unsignedImage,omitempty"`
	StartTime     metav1.Time             `json:"startTime,omitempty"`
	EndTime       metav1.Time             `json:"endTime,omitempty"`
}

type ImageSigningCondition struct {
	Status             v1.ConditionStatus        `json:"status,omitempty"`
	Message            string                    `json:"message,omitempty"`
	LastTransitionTime metav1.Time               `json:"lastTransitionTime,omitempty"`
	Type               ImageSigningConditionType `json:"type,omitempty"`
}

type ImageSigningPhase string

const (
	PhaseRunning   ImageSigningPhase = "Running"
	PhaseCompleted ImageSigningPhase = "Completed"
	PhaseFailed    ImageSigningPhase = "Failed"
)

type ImageSigningConditionType string

const (
	ImageSigningConditionInitialization = "Initialization"
	ImageSigningConditionSigning        = "Signing"
	ImageSigningConditionFinished       = "Finished"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageSigningRequestList is a list of ImageSigningRequest resources
type ImageSigningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImageSigningRequest `json:"items"`
}
