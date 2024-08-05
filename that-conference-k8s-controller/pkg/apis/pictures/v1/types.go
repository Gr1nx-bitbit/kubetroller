package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PodCustomizer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodCustomizerSpec   `json:"spec"`
	Status PodCustomizerStatus `json:"status"`
}

type PodCustomizerSpec struct {
	Promote bool `json:"promote"`
}

type PodCustomizerStatus struct {
	NumPromoted  int64 `json:"numPromoted"`
	NumDestroyed int64 `json:"numDestroyed,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PodCustomizerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PodCustomizer `json:"items"`
}
