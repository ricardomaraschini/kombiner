package controller

import (
	"time"

	"k8s.io/klog/v2"
)

// Option sets an option for a PlacementRequest controller.
type Option func(*options)

// options holds the options for a PlacementRequest controller.
type options struct {
	logger             klog.Logger
	tryToRejectTimeout time.Duration
}

// defaultOptions holds the default options for a PlacementRequest controller.
var defaultOptions = options{
	logger:             klog.NewKlogr(),
	tryToRejectTimeout: 2 * time.Second,
}

// WithLogger sets the logger for the PlacementRequest controller.
func WithLogger(logger klog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithTryToRejectTimeout sets the timeout for trying to reject a PlacementRequest.
// This value is used by the controller only when rejecting a placement request
// is not that important and can fail without causing major consequences. XXX this
// isn't exposed through the API but it might so let's keep this option here.
func WithTryToRejectTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.tryToRejectTimeout = timeout
	}
}
