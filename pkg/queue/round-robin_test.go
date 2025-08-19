package queue

import (
	"context"
	"kombiner/pkg/apis/kombiner/v1alpha1"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundRobinReader_Read(t *testing.T) {
	require := require.New(t)
	for _, tt := range []struct {
		name    string
		configs []ExtendedQueueConfig
		prs     [][]v1alpha1.PlacementRequest
		want    []v1alpha1.PlacementRequest
	}{
		{
			name: "first queue empty",
			configs: []ExtendedQueueConfig{
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "A",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "B",
						Queue: NewPlacementRequestQueue(),
					},
				},
			},
			prs: [][]v1alpha1.PlacementRequest{
				{},
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod3"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod4"},
							},
						},
					},
				},
			},
			want: []v1alpha1.PlacementRequest{
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod3"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod4"},
						},
					},
				},
			},
		},
		{
			name: "all empty",
			configs: []ExtendedQueueConfig{
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "A",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "B",
						Queue: NewPlacementRequestQueue(),
					},
				},
			},
			prs:  [][]v1alpha1.PlacementRequest{},
			want: []v1alpha1.PlacementRequest{},
		},
		{
			name: "return all placement requests",
			configs: []ExtendedQueueConfig{
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "A",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "B",
						Queue: NewPlacementRequestQueue(),
					},
				},
			},
			prs: [][]v1alpha1.PlacementRequest{
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod1"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod2"},
							},
						},
					},
				},
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod3"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod4"},
							},
						},
					},
				},
			},
			want: []v1alpha1.PlacementRequest{
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod1"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod2"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod3"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod4"},
						},
					},
				},
			},
		},
		{
			name: "read max from all queues multiple times",
			configs: []ExtendedQueueConfig{
				{
					MaximumBindings: 1,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "A",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 1,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "B",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 1,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "C",
						Queue: NewPlacementRequestQueue(),
					},
				},
			},
			prs: [][]v1alpha1.PlacementRequest{
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod1"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod2"},
							},
						},
					},
				},
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod3"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod4"},
							},
						},
					},
				},
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod5"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod6"},
							},
						},
					},
				},
			},
			want: []v1alpha1.PlacementRequest{
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod1"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod3"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod5"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod2"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod4"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod6"},
						},
					},
				},
			},
		},
		{
			name: "start reading from the second queue then reset",
			configs: []ExtendedQueueConfig{
				{
					MaximumBindings: 2,
					BindingsRead:    2,
					QueueConfig: QueueConfig{
						Name:  "A",
						Queue: NewPlacementRequestQueue(),
					},
				},
				{
					MaximumBindings: 2,
					BindingsRead:    0,
					QueueConfig: QueueConfig{
						Name:  "B",
						Queue: NewPlacementRequestQueue(),
					},
				},
			},
			prs: [][]v1alpha1.PlacementRequest{
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod1"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod2"},
							},
						},
					},
				},
				{
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod3"},
							},
						},
					},
					{
						Spec: v1alpha1.PlacementRequestSpec{
							Bindings: []v1alpha1.Binding{
								{PodName: "pod4"},
							},
						},
					},
				},
			},
			want: []v1alpha1.PlacementRequest{
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod3"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod4"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod1"},
						},
					},
				},
				{
					Spec: v1alpha1.PlacementRequestSpec{
						Bindings: []v1alpha1.Binding{
							{PodName: "pod2"},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for qidx, prs := range tt.prs {
				for _, pr := range prs {
					tt.configs[qidx].QueueConfig.Queue.Push(&pr)
				}
			}

			ctx := context.Background()
			rr := &RoundRobinReader{configs: tt.configs}
			result := []v1alpha1.PlacementRequest{}
			for pr := rr.Read(ctx); pr != nil; pr = rr.Read(ctx) {
				result = append(result, *pr)
			}

			require.Equal(tt.want, result, "should return expected placement requests")
		})
	}
}

func TestRoundRobinReader_next(t *testing.T) {
	require := require.New(t)
	for _, tt := range []struct {
		name    string
		configs []ExtendedQueueConfig
		want    int
	}{
		{
			name: "first config not exhausted",
			configs: []ExtendedQueueConfig{
				{QueueConfig: QueueConfig{Name: "A"}, MaximumBindings: 2, BindingsRead: 0},
				{QueueConfig: QueueConfig{Name: "B"}, MaximumBindings: 3, BindingsRead: 1},
			},
			want: 0,
		},
		{
			name: "second config available",
			configs: []ExtendedQueueConfig{
				{QueueConfig: QueueConfig{Name: "A"}, MaximumBindings: 2, BindingsRead: 2},
				{QueueConfig: QueueConfig{Name: "B"}, MaximumBindings: 3, BindingsRead: 1},
			},
			want: 1,
		},
		{
			name: "overflown first config",
			configs: []ExtendedQueueConfig{
				{QueueConfig: QueueConfig{Name: "A"}, MaximumBindings: 2, BindingsRead: 8},
				{QueueConfig: QueueConfig{Name: "B"}, MaximumBindings: 3, BindingsRead: 1},
			},
			want: 1,
		},
		{
			name: "all configs exhausted",
			configs: []ExtendedQueueConfig{
				{QueueConfig: QueueConfig{Name: "A"}, MaximumBindings: 1, BindingsRead: 1},
				{QueueConfig: QueueConfig{Name: "B"}, MaximumBindings: 2, BindingsRead: 2},
			},
			want: -1,
		},
		{
			name:    "empty configs",
			configs: []ExtendedQueueConfig{},
			want:    -1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoundRobinReader{configs: tt.configs}
			require.Equal(tt.want, r.next(), "next() should return the expected index")
		})
	}
}

func TestNewRoundRobinReader(t *testing.T) {
	require := require.New(t)
	configs := QueueConfigs{
		{Name: "A", Weight: 2},
		{Name: "B", Weight: 3},
		{Name: "C", Weight: 4},
		{Name: "D", Weight: 2},
		{Name: "E", Weight: 13},
	}

	reader := NewRoundRobinReader(configs)

	expected := []ExtendedQueueConfig{
		{
			QueueConfig:     QueueConfig{Name: "A", Weight: 2},
			MaximumBindings: MinimumBindings,
		},
		{
			QueueConfig:     QueueConfig{Name: "B", Weight: 3},
			MaximumBindings: 15,
		},
		{
			QueueConfig:     QueueConfig{Name: "C", Weight: 4},
			MaximumBindings: 20,
		},
		{
			QueueConfig:     QueueConfig{Name: "D", Weight: 2},
			MaximumBindings: MinimumBindings,
		},
		{
			QueueConfig:     QueueConfig{Name: "E", Weight: 13},
			MaximumBindings: 65,
		},
	}

	rr, ok := reader.(*RoundRobinReader)
	require.True(ok, "reader should be of type RoundRobinReader")
	require.Equal(rr.configs, expected, "expected configs to match")
}
