package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GreetingSpec defines the desired state of Greeting
type GreetingSpec struct {
	// Message is the greeting message to display
	Message string `json:"message,omitempty"`
}

// GreetingStatus defines the observed state of Greeting
type GreetingStatus struct {
	// PodName is the name of the pod created for this greeting
	PodName string `json:"podName,omitempty"`
	// Ready indicates if the greeting pod is ready
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Greeting is the Schema for the greetings API
type Greeting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GreetingSpec   `json:"spec,omitempty"`
	Status GreetingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GreetingList contains a list of Greeting
type GreetingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Greeting `json:"items"`
}
