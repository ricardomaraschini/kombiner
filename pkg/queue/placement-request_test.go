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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kombiner/pkg/apis/kombiner/v1alpha1"
)

func TestPlacementRequestQueue(t *testing.T) {
	assert := assert.New(t)

	queue := NewPlacementRequestQueue()
	for i := range 10 {
		sub := time.Duration(i) * time.Hour * -1
		pr := &v1alpha1.PlacementRequest{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{
					Time: metav1.Now().Time.Add(sub),
				},
			},
		}
		queue.Push(pr)
	}

	var last *time.Time
	for range 10 {
		pr := queue.Pop()
		assert.NotNil(pr, "expected a placement request but got nil")

		current := &pr.CreationTimestamp.Time
		if last == nil {
			last = current
			continue
		}

		assert.Less(*last, *current, "expected placement request to be after the last one")
		last = current
	}
}

func TestPlacementRequestPushHandlers(t *testing.T) {
	assert := assert.New(t)

	counter := 0
	pushHandler := func() {
		counter++
	}

	queue := NewPlacementRequestQueue()
	queue.AddPushHandler(pushHandler)
	for range 10 {
		queue.Push(&v1alpha1.PlacementRequest{})
	}

	assert.Equal(10, counter, "expected push handler to be called 10 times")
}
