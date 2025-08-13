/*
Copyright The Kubernetes Authors.

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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	configapi "kombiner/pkg/apis/config/v1alpha1"
)

func TestValidate(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := configapi.AddToScheme(testScheme); err != nil {
		t.Fatal(err)
	}
	if err := clientgoscheme.AddToScheme(testScheme); err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		cfg     *configapi.Configuration
		wantErr field.ErrorList
	}{
		"empty": {
			cfg: &configapi.Configuration{},
			wantErr: field.ErrorList{
				&field.Error{
					Type:  field.ErrorTypeRequired,
					Field: "queues",
				},
			},
		},
		"invalid queue scheduler name": {
			cfg: &configapi.Configuration{
				Queues: []configapi.Queue{
					{
						SchedulerName: "",
						Weight:        1,
						MaxSize:       1,
					},
				},
			},
			wantErr: field.ErrorList{
				field.Required(field.NewPath("queues").Index(0).Child("schedulerName"), nonEmptyErrStr),
			},
		},
		"invalid queue weight": {
			cfg: &configapi.Configuration{
				Queues: []configapi.Queue{
					{
						SchedulerName: "default-scheduler",
						Weight:        -1,
						MaxSize:       1,
					},
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("queues").Index(0).Child("weight"), "", mustBePositiveIntegerErrStr),
			},
		},
		"invalid queue maxsize": {
			cfg: &configapi.Configuration{
				Queues: []configapi.Queue{
					{
						SchedulerName: "default-scheduler",
						Weight:        1,
						MaxSize:       0,
					},
				},
			},
			wantErr: field.ErrorList{
				field.Invalid(field.NewPath("queues").Index(0).Child("maxSize"), "", mustBePositiveIntegerErrStr),
			},
		},
		// TODO(ingvagabund):
		// more tests:
		// - no duplicates in enabled/disabled list of plugins (for both queue based and cluster wide)
		// - no intersection of plugin names in enabled and disabled lists (for both queue based and cluster wide)
		// - enabled/disabled plugins must be known names (for both queue based and cluster wide)
		// - no two queues have the scheduler schedulerName
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.wantErr, validate(tc.cfg), cmpopts.IgnoreFields(field.Error{}, "BadValue", "Detail")); diff != "" {
				t.Errorf("Unexpected returned error (-want,+got):\n%s", diff)
			}
		})
	}
}
