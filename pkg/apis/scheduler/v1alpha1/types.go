package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlacementRequestBinderArgs defines the configuration available for the
// placement request binder plugin.
type PlacementRequestBinderArgs struct {
	metav1.TypeMeta

	// Timeout defines how long the scheduler waits until it gives up on a
	// placement request.
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}
