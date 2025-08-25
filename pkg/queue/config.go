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

package queue

import (
	"fmt"

	"kombiner/pkg/apis/config/v1alpha1"
)

// QueueConfig defines the configuration for a queue in the queue iterator. It
// includes the name of the queue, its weight, and the actual queue. The Weight
// determines how often the queue will be processed in each iteration and it is
// proportional to the sum of all weights provided for the QueueIterator.
type QueueConfig struct {
	v1alpha1.Queue

	QueueRef *PlacementRequestQueue
}

// Validate checks the QueueConfig for correctness. We ensure that the queue
// has a name, its weight is greater than zero and that we have a valid pointer
// to a PlacementRequestQueue.
func (c *QueueConfig) Validate() error {
	if c.SchedulerName == "" {
		return fmt.Errorf("queue name cannot be empty")
	}
	if c.Weight == 0 {
		return fmt.Errorf("queue weight must be greater than zero")
	}
	if c.MaxSize == 0 {
		return fmt.Errorf("queue max size must be greater than zero")
	}
	if c.QueueRef == nil {
		return fmt.Errorf("queue reference cannot be nil")
	}
	return nil
}

// QueueConfigs is a list of QueueConfig objects. This struct is useful for
// batch operations over multiple queues.
type QueueConfigs []QueueConfig

// ToMap converts the QueueConfigs to a map where the key is the queue name
// and the value is the PlacementRequestQueue. Multiple QueueConfigs for the
// same queue name will overwrite each other.
func (c QueueConfigs) ToMap() map[string]QueueConfig {
	m := make(map[string]QueueConfig, len(c))
	for _, cfg := range c {
		m[cfg.SchedulerName] = cfg
	}
	return m
}

// AddPushHandlers adds the same push handler to all queues within a queue
// configs slice.
func (c *QueueConfigs) AddPushHandler(handler func()) {
	for _, config := range *c {
		config.QueueRef.AddPushHandler(handler)
	}
}

// Validate checks the QueueConfigs for correctness. We ensure that each queue
// has the mandatory fields and that they all have different names.
func (c QueueConfigs) Validate() error {
	seen := map[string]bool{}
	for _, cfg := range c {
		if _, ok := seen[cfg.SchedulerName]; ok {
			return fmt.Errorf(
				"duplicate config for scheduler %q",
				cfg.SchedulerName,
			)
		}
		seen[cfg.SchedulerName] = true
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf(
				"invalid config for scheduler %q: %w",
				cfg.SchedulerName,
				err,
			)
		}
	}
	return nil
}

// QueueConfigFromV1Alpha1Config parses the controller configuration directly
// into a QueueConfigs object. No validation is performed at this stage.
func QueueConfigFromV1Alpha1Config(raw v1alpha1.Configuration) QueueConfigs {
	configs := QueueConfigs{}
	for _, config := range raw.Queues {
		configs = append(
			configs, QueueConfig{
				Queue:    config,
				QueueRef: NewPlacementRequestQueue(),
			},
		)
	}
	return configs
}
