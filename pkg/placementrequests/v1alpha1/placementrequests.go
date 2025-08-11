package v1alpha1

import (
	"errors"
	"fmt"

	"kombiner/pkg/apis/v1alpha1"
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
