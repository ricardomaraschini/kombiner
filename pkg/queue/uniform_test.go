package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniformReader_next(t *testing.T) {
	assert := assert.New(t)

	configs := QueueConfigs{
		{
			Name:   "scheduler-1",
			Weight: 7,
			Queue:  NewPlacementRequestQueue(),
		},
		{
			Name:   "scheduler-2",
			Weight: 2,
			Queue:  NewPlacementRequestQueue(),
		},
		{
			Name:   "scheduler-3",
			Weight: 1,
			Queue:  NewPlacementRequestQueue(),
		},
	}

	reader := NewUniformReader(configs)
	uniform, ok := reader.(*UniformReader)
	require.True(t, ok, "reader should be of type UniformReader")

	counters := map[string]int{}
	iterations := 10000000
	for range iterations {
		next := uniform.next(configs)
		counters[configs[next].Name]++
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
