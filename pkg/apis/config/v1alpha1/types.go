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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:defaulter-gen=true
// +kubebuilder:object:root=true

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Configuration is the Schema for the kombinerconfigurations API
type Configuration struct {
	metav1.TypeMeta `json:",inline"`

	// Queues provides configuration for individual queues
	Queues []Queue `json:"queues"`

	// Plugins captures a configuration for cluster wide validation
	// +optional
	Plugins Plugins `json:"plugins,omitempty"`
}

// Queue represents a scheduler queue configuration.
type Queue struct {
	// SchedulerName targets placement requests from a specific scheduler (or a profile)
	SchedulerName string `json:"schedulerName"`

	// Weight determines how often a scheduler's placement requests get reconciled
	// compared to other schedulers
	Weight int `json:"weight"`

	// MaxSize bounds the maximum size of a placement requests.
	// I.e. how many pod-to-node assignments can be listed in a placement request.
	MaxSize int `json:"maxSize"`

	// Plugins configures a list of enabled/disabled plugins for a scheduler
	// E.g. the scheduling framework provides many native plugins. Yet, some
	// profiles might disable plugins enabled by default. Configuration
	// provided her makes the kombiner controller know which plugins
	// need to be validated before final admission.
	Plugins Plugins `json:"plugins"`
}

// Plugins represents plugin configuration at either cluster or queue level.
type Plugins struct {
	// Validate carries a list of enabled/disabled validate extension points
	Validate PluginSet `json:"validate"`
}

// PluginSet contains the list of enabled and disabled plugins
type PluginSet struct {
	// Enabled configures a list of enabled plugins
	Enabled []string `json:"enabled,omitempty"`

	// Disabled configures a list of disabled plugins
	Disabled []string `json:"disabled,omitempty"`
}
