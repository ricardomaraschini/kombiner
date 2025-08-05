package controller

import "k8s.io/klog/v2"

// Option sets an option for a PlacementRequest controller.
type Option func(*options)

// options holds the options for a PlacementRequest controller.
type options struct {
	logger klog.Logger
}

// defaultOptions holds the default options for a PlacementRequest controller.
var defaultOptions = options{
	logger: klog.NewKlogr(),
}

// WithLogger sets the logger for the PlacementRequest controller.
func WithLogger(logger klog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
