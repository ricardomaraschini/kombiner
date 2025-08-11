package composition

import (
	"fmt"
	"slices"

	"kombiner/pkg/apis/v1alpha1"
)

// PlacementRequest purpose is to provide auxiliar methods to an existing
// v1alpha1.PlacementRequest.
type PlacementRequest struct {
	*v1alpha1.PlacementRequest
}

// SetBindingFailure helps when adding a bind failure condition to the
// PlacementRequest.
func (pr *PlacementRequest) SetBindingFailure(bind v1alpha1.Binding, reason, msg string) {
	pr.Status.Bindings = append(
		pr.Status.Bindings,
		v1alpha1.PlacementRequestBindingResult{
			Binding: bind,
			Result:  v1alpha1.PlacementRequestResultFailure,
			Reason:  reason,
			Message: msg,
		},
	)
}

// SetBindingSuccess helps when setting a successful binding condition for a
// given v1alpha1.Binding.
func (pr *PlacementRequest) SetBindingSuccess(bind v1alpha1.Binding) {
	pr.Status.Bindings = append(
		pr.Status.Bindings,
		v1alpha1.PlacementRequestBindingResult{
			Binding: bind,
			Result:  v1alpha1.PlacementRequestResultSuccess,
			Reason:  "Binding successful",
			Message: "The pod was successfully bound to the node",
		},
	)
}

// AssessResult helps when assessing the result of a PlacementRequest bind
// attempt. It uses the status to verify if all bindings succeeded or not.
// XXX we do not yet support the AllOrNothing policy so this will need to
// change later down the line (i.e. a single failure unders such policy will
// result in a full failure).
func (pr *PlacementRequest) AssessResult() {
	successes := slices.DeleteFunc(
		pr.Status.Bindings,
		func(b v1alpha1.PlacementRequestBindingResult) bool {
			return b.Result != v1alpha1.PlacementRequestResultSuccess
		},
	)

	switch {
	case len(successes) == 0:
		pr.SetResult(v1alpha1.PlacementRequestResultFailure, "All bindings failed")
	case len(successes) < len(pr.Status.Bindings):
		pr.SetResult(v1alpha1.PlacementRequestResultPartialSuccess, "Some bindings succeeded")
	default:
		pr.SetResult(v1alpha1.PlacementRequestResultSuccess, "All bindings succeeded")
	}
}

// SetResult goal is to set the result and the message of the PlacementRequest.
func (pr *PlacementRequest) SetResult(res v1alpha1.PlacementRequestResult, msg string) {
	pr.Status.Result, pr.Status.Message = res, msg
}

// Validate is used to validate a PlacementRequest object. If invalid then this
// helps in seting the correct Result and Message status properties.
func (pr *PlacementRequest) Valid() bool {
	if len(pr.Spec.Bindings) == 0 {
		pr.SetResult(v1alpha1.PlacementRequestResultRejected, "Placement request has no bindings")
		return false
	}

	if pr.Spec.Policy != v1alpha1.PlacementRequestPolicyLenient {
		message := fmt.Sprintf("Unsupported policy: %s", pr.Spec.Policy)
		pr.SetResult(v1alpha1.PlacementRequestResultRejected, message)
		return false
	}

	return true
}
