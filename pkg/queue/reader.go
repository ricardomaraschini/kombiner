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
	"context"
	"kombiner/pkg/apis/kombiner/v1alpha1"
)

// Reader function is used to abstract the multiple queue readers we have
// implemented. The role of a reader is to determine the order in which
// messages should be read from the provided queues and return the next
// PlacementRequest to be processed. The readers are expected to return
// nil if there is nothing else to be read in a given moment.
type Reader interface {
	Read(context.Context) *v1alpha1.PlacementRequest
}

// ReaderFactory is a funtion that return a queue reader for a list of
// provided queues.
type ReaderFactory func(QueueConfigs) Reader
