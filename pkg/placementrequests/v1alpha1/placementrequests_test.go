package v1alpha1

import (
	"testing"

	"kombiner/pkg/apis/kombiner/v1alpha1"

	"github.com/stretchr/testify/require"
)

func TestAssessResult(t *testing.T) {
	tests := []struct {
		name            string
		pr              *v1alpha1.PlacementRequest
		expectedResult  v1alpha1.PlacementRequestResult
		expectedMessage string
	}{
		{
			name: "all bindings succeeded",
			pr: &v1alpha1.PlacementRequest{
				Status: v1alpha1.PlacementRequestStatus{
					Bindings: []v1alpha1.PlacementRequestBindingResult{
						{Result: v1alpha1.PlacementRequestResultSuccess},
						{Result: v1alpha1.PlacementRequestResultSuccess},
					},
				},
			},
			expectedResult:  v1alpha1.PlacementRequestResultSuccess,
			expectedMessage: "All bindings succeeded",
		},
		{
			name: "all bindings failed",
			pr: &v1alpha1.PlacementRequest{
				Status: v1alpha1.PlacementRequestStatus{
					Bindings: []v1alpha1.PlacementRequestBindingResult{
						{Result: v1alpha1.PlacementRequestResultFailure},
						{Result: v1alpha1.PlacementRequestResultFailure},
					},
				},
			},
			expectedResult:  v1alpha1.PlacementRequestResultFailure,
			expectedMessage: "All bindings failed",
		},
		{
			name: "partial success",
			pr: &v1alpha1.PlacementRequest{
				Status: v1alpha1.PlacementRequestStatus{
					Bindings: []v1alpha1.PlacementRequestBindingResult{
						{Result: v1alpha1.PlacementRequestResultSuccess},
						{Result: v1alpha1.PlacementRequestResultFailure},
					},
				},
			},
			expectedResult:  v1alpha1.PlacementRequestResultPartialSuccess,
			expectedMessage: "1 of 2 bindings succeeded",
		},
		{
			name: "no bindings",
			pr: &v1alpha1.PlacementRequest{
				Status: v1alpha1.PlacementRequestStatus{
					Bindings: []v1alpha1.PlacementRequestBindingResult{},
				},
			},
			expectedResult:  v1alpha1.PlacementRequestResultRejected,
			expectedMessage: "No bindings",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, message := AssessResult(test.pr)
			require.Equal(t, test.expectedResult, result)
			require.Equal(t, test.expectedMessage, message)
		})
	}
}
