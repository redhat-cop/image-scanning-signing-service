package images

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
