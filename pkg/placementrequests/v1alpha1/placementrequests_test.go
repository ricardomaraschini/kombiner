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
