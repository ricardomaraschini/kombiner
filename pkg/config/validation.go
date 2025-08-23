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

package config

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	configapi "kombiner/pkg/apis/config/v1alpha1"
)

var (
	queuesPath = field.NewPath("queues")

	nonEmptyErrStr              = "must be non-empty"
	mustBePositiveIntegerErrStr = "must be a positive integer"
)

func validate(c *configapi.Configuration) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateQueues(c)...)
	return allErrs
}

func validateQueues(c *configapi.Configuration) field.ErrorList {
	var allErrs field.ErrorList

	if len(c.Queues) == 0 {
		return append(allErrs, field.Required(queuesPath, nonEmptyErrStr))
	}

	for idx, queue := range c.Queues {
		if queue.SchedulerName == "" {
			allErrs = append(allErrs, field.Required(queuesPath.Index(idx).Child("schedulerName"), nonEmptyErrStr))
		}
		if queue.Weight < 1 {
			allErrs = append(allErrs, field.Invalid(queuesPath.Index(idx).Child("weight"), queue.Weight, mustBePositiveIntegerErrStr))
		}
		if queue.MaxSize < 1 {
			allErrs = append(allErrs, field.Invalid(queuesPath.Index(idx).Child("maxSize"), queue.MaxSize, mustBePositiveIntegerErrStr))
		}
	}

	return allErrs
}
