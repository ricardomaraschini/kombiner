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
