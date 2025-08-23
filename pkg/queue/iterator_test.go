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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configv1alpha1 "kombiner/pkg/apis/config/v1alpha1"
	"kombiner/pkg/apis/kombiner/v1alpha1"
)

func TestQueueIteratorFairness(t *testing.T) {
	assert := assert.New(t)

	configs := QueueConfigs{
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-1",
				Weight:        35,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-2",
				Weight:        35,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-3",
				Weight:        10,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-4",
				Weight:        5,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-5",
				Weight:        5,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-6",
				Weight:        3,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-7",
				Weight:        3,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-8",
				Weight:        3,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-9",
				Weight:        1,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
	}

	iterator, err := NewQueueIterator(configs)
	assert.NoError(err, "error creating iterator")

	go iterator.Run(context.Background())

	for i := range len(configs) {
		go func(idx int) {
			name := fmt.Sprintf("scheduler-%d", idx%len(configs)+1)
			for range 1000000 {
				configs[idx].QueueRef.Push(
					&v1alpha1.PlacementRequest{
						Spec: v1alpha1.PlacementRequestSpec{
							SchedulerName: name,
						},
					},
				)
			}
		}(i)
	}

	counters := map[string]int{}
	for range 1000000 {
		ticker := time.NewTicker(time.Second)
		select {
		case pr := <-iterator.Next:
			counters[pr.Spec.SchedulerName]++
			ticker.Stop()
		case <-ticker.C:
			t.Fatalf("timeout waiting for requests")
		}
	}

	percentage := map[string]int{}
	for name, c := range counters {
		percentage[name] = int(float64(c) / 1000000 * 100)
	}

	// ballpark here, we expect the first queue to be selected 35% of the time,
	// the second queue 35% of the time, the third queue 10% of the time, the
	// fourth queue 5% of the time, the fifth queue 5% of the time, the sixth
	// queue 3% of the time, the seventh queue 3% of the time, the eighth queue
	// 3% of the time and the ninth queue 1% of the time. we give them a 2% margin
	// of error.
	assert.GreaterOrEqual(percentage["scheduler-1"], 33, "scheduler-1 should be selected at least 33% of the time")
	assert.LessOrEqual(percentage["scheduler-1"], 37, "scheduler-1 should be selected at most 37% of the time")

	assert.GreaterOrEqual(percentage["scheduler-2"], 33, "scheduler-2 should be selected at least 33% of the time")
	assert.LessOrEqual(percentage["scheduler-2"], 37, "scheduler-2 should be selected at most 37% of the time")

	assert.GreaterOrEqual(percentage["scheduler-3"], 8, "scheduler-3 should be selected at least 8% of the time")
	assert.LessOrEqual(percentage["scheduler-3"], 12, "scheduler-3 should be selected at most 12% of the time")

	assert.GreaterOrEqual(percentage["scheduler-4"], 3, "scheduler-4 should be selected at least 3% of the time")
	assert.LessOrEqual(percentage["scheduler-4"], 7, "scheduler-4 should be selected at most 7% of the time")

	assert.GreaterOrEqual(percentage["scheduler-5"], 3, "scheduler-5 should be selected at least 3% of the time")
	assert.LessOrEqual(percentage["scheduler-5"], 7, "scheduler-5 should be selected at most 7% of the time")

	assert.GreaterOrEqual(percentage["scheduler-6"], 1, "scheduler-6 should be selected at least 1% of the time")
	assert.LessOrEqual(percentage["scheduler-6"], 5, "scheduler-6 should be selected at most 5% of the time")

	assert.GreaterOrEqual(percentage["scheduler-7"], 1, "scheduler-7 should be selected at least 1% of the time")
	assert.LessOrEqual(percentage["scheduler-7"], 5, "scheduler-7 should be selected at most 5% of the time")

	assert.GreaterOrEqual(percentage["scheduler-8"], 1, "scheduler-8 should be selected at least 1% of the time")
	assert.LessOrEqual(percentage["scheduler-8"], 5, "scheduler-8 should be selected at most 5% of the time")

	assert.GreaterOrEqual(percentage["scheduler-9"], 0, "scheduler-9 should be selected at least 0% of the time")
	assert.LessOrEqual(percentage["scheduler-9"], 3, "scheduler-9 should be selected at most 3% of the time")
}

func TestQueueIteratorReadBeforeWrite(t *testing.T) {
	assert := assert.New(t)

	configs := QueueConfigs{
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-1",
				Weight:        1,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
	}

	iterator, err := NewQueueIterator(configs)
	assert.NoError(err, "error creating iterator")

	go iterator.Run(context.Background())

	go func() {
		time.Sleep(time.Second)
		configs[0].QueueRef.Push(&v1alpha1.PlacementRequest{})
	}()

	ticker := time.NewTicker(2 * time.Second)
	select {
	case <-iterator.Next:
		// we expect to get a request here.
	case <-ticker.C:
		t.Fatalf("timeout waiting for requests")
	}
}

func TestQueueIteratorMultipleConcurrent(t *testing.T) {
	require := require.New(t)

	configs := QueueConfigs{
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-1",
				Weight:        35,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-2",
				Weight:        35,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-3",
				Weight:        10,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
	}

	iterator, err := NewQueueIterator(configs)
	require.NoError(err, "error creating iterator")

	go iterator.Run(context.Background())

	for i := range 100 {
		go func(idx int) {
			config := configs[idx%len(configs)]
			for range 1000 {
				sleep := rand.Intn(10)
				time.Sleep(time.Duration(sleep) * time.Millisecond)
				config.QueueRef.Push(
					&v1alpha1.PlacementRequest{
						Spec: v1alpha1.PlacementRequestSpec{
							SchedulerName: config.SchedulerName,
						},
					},
				)
			}
		}(i)
	}

	for range 100000 {
		ticker := time.NewTicker(time.Second)
		select {
		case <-iterator.Next:
			ticker.Stop()
		case <-ticker.C:
			t.Fatalf("timeout waiting for requests")
		}
	}

	ticker := time.NewTicker(time.Second)
	select {
	case <-iterator.Next:
		t.Fatal("expected no more requests, but got one")
	case <-ticker.C:
		ticker.Stop()
	}
}

func TestQueueContextCancellation(t *testing.T) {
	require := require.New(t)

	configs := QueueConfigs{
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-1",
				Weight:        10,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
		{
			Queue: configv1alpha1.Queue{
				SchedulerName: "scheduler-2",
				Weight:        10,
				MaxSize:       100,
			},
			QueueRef: NewPlacementRequestQueue(),
		},
	}

	iterator, err := NewQueueIterator(configs)
	require.NoError(err, "error creating iterator")

	ctx, cancel := context.WithCancel(context.Background())
	runEnded := make(chan struct{})
	go func() {
		iterator.Run(ctx)
		close(runEnded)
	}()

	for i := range len(configs) {
		go func(idx int) {
			for {
				sleep := time.Duration(rand.Intn(100))
				time.Sleep(sleep * time.Millisecond)
				configs[i].QueueRef.Push(&v1alpha1.PlacementRequest{})
			}
		}(i)
	}

	go func() {
		time.Sleep(3 * time.Second)
		cancel()
	}()

	ticker := time.NewTicker(5 * time.Second)
	for range iterator.Next {
		select {
		case <-ticker.C:
			t.Fatalf("timeout waiting for context cancellation")
		default:
		}
	}
	ticker.Stop()

	ticker = time.NewTicker(time.Second)
	select {
	case <-runEnded:
	case <-ticker.C:
		t.Fatalf("timeout waiting for iterator to finish")
	}
}
