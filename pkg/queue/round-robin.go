package queue

import (
	"context"
	"kombiner/pkg/apis/kombiner/v1alpha1"
	"math"
	"slices"
)

// MinimumBindings is the very minimum binds we will ensure to the queue with
// the least weight. Each bind in a PlacementRequest is counted towards this
// amount.
const MinimumBindings = 10

// ExtendedQueueConfig extends a QueueConfig to also contain a property holding
// how many binds are allowed for a queue and a counter for how many binds were
// already read.
type ExtendedQueueConfig struct {
	QueueConfig
	MaximumBindings int
	BindingsRead    int
}

type RoundRobinReader struct {
	configs []ExtendedQueueConfig
}

// Read keeps reading from the same queue until it is empty or we reached the
// max of bindings defined per queue. The maximum number of bindings is
// relative to the weight of each queue. The queue with the lowest weight
// receives MinimumBindings.
func (r *RoundRobinReader) Read(ctx context.Context) *v1alpha1.PlacementRequest {
	// if the queues are empty at this stage we return nil as there is
	// nothing else to read. this is also our stop condition for the
	// recursive calls.
	if r.empty() {
		return nil
	}

	var index int
	if index = r.next(); index < 0 {
		// if we can't find a queue from which to read and we know they
		// aren't empty then we can reset and start over.
		r.reset()
		if index = r.next(); index < 0 {
			// this should never happen
			panic("no queues to read from but not all are empty")
		}
	}

	if pr := r.configs[index].Queue.Pop(); pr != nil {
		r.configs[index].BindingsRead += len(pr.Spec.Bindings)
		return pr
	}

	// we haven't found nothing to read on this queue so we aren't supposed
	// to keep waiting. we set the BindingsRead for the queue to the max to
	// indicate we don't want to read from it anymore and then call this
	// function recursively to try the next queue.
	r.configs[index].BindingsRead = r.configs[index].MaximumBindings
	return r.Read(ctx)
}

// empty returns true if all queues are empty.
func (r *RoundRobinReader) empty() bool {
	for _, cfg := range r.configs {
		if cfg.Queue.Len() > 0 {
			return false
		}
	}
	return true
}

// reset resets the BindingsRead counter for all queues. This is useful when we
// want to start reading from the beginning again.
func (r *RoundRobinReader) reset() {
	for i, cfg := range r.configs {
		cfg.BindingsRead = 0
		r.configs[i] = cfg
	}
}

// next function is to return the next queue from where we should read. this
// function iterates over the configs and finds one for which we haven't yet
// exhausted the maximum number of bindings. If all queues are exhausted
// it returns -1.
func (r *RoundRobinReader) next() int {
	for i, cfg := range r.configs {
		if cfg.BindingsRead < cfg.MaximumBindings {
			return i
		}
	}
	return -1
}

// NewRoundRobinReader creates a new RoundRobinReader with the provided queue
// configurations. Each queue configuration is extended with a default maximum
// number of bindings, which must be equal or greater than MinimumBindings.
// The lighter queue configuration receives MinimumBindings and the rest are
// calculated based on the weight of each queue relative to the lightest one.
// This function expects the configuration to be properly sanitized before
// entering here, it bravely runs away with a panic with invalid data.
func NewRoundRobinReader(configs QueueConfigs) Reader {
	lighter := slices.MinFunc(
		configs,
		func(a, b QueueConfig) int {
			return int(a.Weight) - int(b.Weight)
		},
	)

	// this is a sanity check, we can't have queues with zeroed out
	// weights as this makes no sense in the context. We are dividing
	// by the lowest weight but as we convert it to a float to do so
	// we end up obtaining a +Inf value, and that is no bueno.
	if lighter.Weight == 0 {
		panic("queue with zero weight provided")
	}

	extended := make([]ExtendedQueueConfig, len(configs))
	for i, cfg := range configs {
		multiplier := float64(cfg.Weight) / float64(lighter.Weight)
		maxbindings := math.Ceil(multiplier * float64(MinimumBindings))
		extended[i] = ExtendedQueueConfig{
			QueueConfig:     cfg,
			MaximumBindings: int(maxbindings),
		}
	}

	return &RoundRobinReader{
		configs: extended,
	}
}
