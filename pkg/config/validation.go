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
