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
