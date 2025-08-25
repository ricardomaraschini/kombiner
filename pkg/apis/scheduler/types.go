/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scheduler

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   "kubescheduler.config.k8s.io",
	Version: runtime.APIVersionInternal,
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlacementRequestBinderArgs defines the configuration available for the
// placement request binder plugin.
type PlacementRequestBinderArgs struct {
	metav1.TypeMeta `json:",inline"`

	// Timeout defines how long the scheduler waits until it gives up on a
	// placement request.
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// SetDefaults is used to set default values for the scheduler plugin
// configuration.
func SetDefaults(pr *PlacementRequestBinderArgs) {
	if pr.Timeout == nil {
		pr.Timeout = &metav1.Duration{
			Duration: time.Minute,
		}
	}
}
