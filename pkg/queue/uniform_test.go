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
	"kombiner/pkg/apis/config/v1alpha1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniformReader_next(t *testing.T) {
	assert := assert.New(t)

	configs := QueueConfigs{
		{
			Queue: v1alpha1.Queue{
				SchedulerName: "scheduler-1",
				Weight:        7,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: v1alpha1.Queue{
				SchedulerName: "scheduler-2",
				Weight:        2,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: v1alpha1.Queue{
				SchedulerName: "scheduler-3",
				Weight:        1,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
	}

	reader := NewUniformReader(configs)
	uniform, ok := reader.(*UniformReader)
	require.True(t, ok, "reader should be of type UniformReader")

	counters := map[string]int{}
	iterations := 10000000
	for range iterations {
		next := uniform.next(configs)
		counters[configs[next].SchedulerName]++
	}

	percentage := map[string]int{}
	for name, c := range counters {
		percentage[name] = int(float64(c) / float64(iterations) * 100)
	}

	// ballpark here, we expect the first queue to be selected 70% of the time,
	// the second queue 20% of the time and the third queue 10% of the time. we
	// give them a 2% margin of error.
	assert.GreaterOrEqual(percentage["scheduler-1"], 68, "scheduler-1 should be selected at least 68% of the time")
	assert.LessOrEqual(percentage["scheduler-1"], 72, "scheduler-1 should be selected at most 72% of the time")

	assert.GreaterOrEqual(percentage["scheduler-2"], 18, "scheduler-2 should be selected at least 18% of the time")
	assert.LessOrEqual(percentage["scheduler-2"], 22, "scheduler-2 should be selected at most 22% of the time")

	assert.GreaterOrEqual(percentage["scheduler-3"], 8, "scheduler-3 should be selected at least 8% of the time")
	assert.LessOrEqual(percentage["scheduler-3"], 12, "scheduler-3 should be selected at most 12% of the time")
}
