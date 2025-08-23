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

package v1alpha1

import (
	"errors"
	"fmt"

	"kombiner/pkg/apis/kombiner/v1alpha1"
)

// Validate makes some basic validation checks on the PlacementRequest. This
// function returns an error if the PlacementRequest is not valid. The error
// can be used to set the status of the PlacementRequest.
func Validate(pr *v1alpha1.PlacementRequest) error {
	// if the placement request has no bindings, it is not valid.
	if len(pr.Spec.Bindings) == 0 {
		return errors.New("the placement request has no bindings")
	}

	// the current implementation only supports the lenient policy, even
	// though we still allow these objects to be created with the
	// AllOrNothing policy.
	if pr.Spec.Policy != v1alpha1.PlacementRequestPolicyLenient {
		return fmt.Errorf("unsupported policy: %s", pr.Spec.Policy)
	}

	return nil
}

// SetPodBindingFailure is a sugar coated version of the SetPodBindingResult
// function. This allows for a shorter function call when setting a failure.
func SetPodBindingFailure(pr *v1alpha1.PlacementRequest, bind v1alpha1.Binding, reason, msg string) {
	SetPodBindingResult(pr, bind, v1alpha1.PlacementRequestResultFailure, reason, msg)
}

// SetPodBindingSuccess is a sugar coated version of the SetPodBindingResult
// function. This allows for a shorter function call when setting a success.
func SetPodBindingSuccess(pr *v1alpha1.PlacementRequest, bind v1alpha1.Binding, reason, msg string) {
	SetPodBindingResult(pr, bind, v1alpha1.PlacementRequestResultSuccess, reason, msg)
}

// SetPodBindingResult is a helper function that implements the logic of
// setting a single pod binding result in the pod placement status.
// XXX this function needs to be optimized for performance. This may
// potentially involve changing the API type and that will require more
// discussion.
func SetPodBindingResult(
	pr *v1alpha1.PlacementRequest,
	bind v1alpha1.Binding,
	result v1alpha1.PlacementRequestResult,
	reason, msg string,
) {
	for i, b := range pr.Status.Bindings {
		if b.Binding.PodUID == bind.PodUID {
			pr.Status.Bindings[i] = v1alpha1.PlacementRequestBindingResult{
				Binding: bind, Result: result, Reason: reason, Message: msg,
			}
			return
		}
	}
	pr.Status.Bindings = append(
		pr.Status.Bindings,
		v1alpha1.PlacementRequestBindingResult{
			Binding: bind, Result: result, Reason: reason, Message: msg,
		},
	)
}

// AssessResult role is to assess, based on the placement request status, if
// it was successful or not. This function returns the result and a human
// readable message.
func AssessResult(pr *v1alpha1.PlacementRequest) (v1alpha1.PlacementRequestResult, string) {
	if len(pr.Status.Bindings) == 0 {
		return v1alpha1.PlacementRequestResultRejected, "No bindings"
	}

	successes := 0
	for _, b := range pr.Status.Bindings {
		if b.Result == v1alpha1.PlacementRequestResultSuccess {
			successes++
		}
	}

	switch {
	case successes == 0:
		return v1alpha1.PlacementRequestResultFailure, "All bindings failed"
	case successes == len(pr.Status.Bindings):
		return v1alpha1.PlacementRequestResultSuccess, "All bindings succeeded"
	default:
		msg := fmt.Sprintf(
			"%d of %d bindings succeeded", successes, len(pr.Status.Bindings),
		)
		return v1alpha1.PlacementRequestResultPartialSuccess, msg
	}
}
