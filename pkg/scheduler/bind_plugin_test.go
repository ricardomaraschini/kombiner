package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2/ktesting"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	st "k8s.io/kubernetes/pkg/scheduler/testing"

	"kombiner/pkg/apis/v1alpha1"
	"kombiner/pkg/generated/clientset/versioned/fake"
)

func TestBindPluginSchedulerNameSet(t *testing.T) {
	tests := []struct {
		name                  string
		podUID                types.UID
		schedulerName         string
		expectedSchedulerName string
		skipPRPulling         bool
		injectErr             error
	}{
		{
			name:                  "default scheduler",
			podUID:                "8d9f8c5e-13a3-4b7d-8d7e-4c23b19c1f74",
			expectedSchedulerName: corev1.DefaultSchedulerName,
		},
		{
			name:                  "custom scheduler",
			podUID:                "8d9f8c5e-13a3-4b7d-8d7e-4c23b19c1f74",
			schedulerName:         "custom-scheduler",
			expectedSchedulerName: "custom-scheduler",
		},
		{
			name:                  "no pod uid",
			schedulerName:         "custom-scheduler",
			expectedSchedulerName: "custom-scheduler",
			skipPRPulling:         true,
			injectErr:             fmt.Errorf("pod ns/foo is missing its UID"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			client := fake.NewSimpleClientset()

			testPod := st.MakePod().Name("foo").Namespace("ns").Obj()
			testPod.UID = tt.podUID
			testPod.Spec.SchedulerName = tt.schedulerName

			var bindStatus *framework.Status
			bindDoneChan := make(chan struct{})
			// run the binding step in parallel so the created PR can be inspected and updated
			go func() {
				binder := &BindPlugin{client: client}
				bindStatus = binder.Bind(ctx, nil, testPod, "testNode")
				bindDoneChan <- struct{}{}
			}()

			if !tt.skipPRPulling {
				prClient := client.KombinerV1alpha1().PlacementRequests(testPod.Namespace)
				if err := wait.PollUntilContextTimeout(
					ctx, 50*time.Millisecond, time.Second, true,
					func(ctx context.Context) (bool, error) {
						pr, err := prClient.Get(ctx, string(testPod.UID), metav1.GetOptions{})
						if err != nil {
							if errors.IsNotFound(err) {
								return false, nil
							}
							return false, err
						}
						pr.Status = v1alpha1.PlacementRequestStatus{
							Result: v1alpha1.PlacementRequestResultSuccess,
						}
						_, err = prClient.Update(ctx, pr, metav1.UpdateOptions{})
						if err != nil {
							return false, err
						}

						if diff := cmp.Diff(tt.expectedSchedulerName, pr.Spec.SchedulerName); diff != "" {
							t.Errorf("got different schedulerName (-want, +got): %s", diff)
						}

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			}

			<-bindDoneChan
			if got := bindStatus.AsError(); (tt.injectErr != nil) != (got != nil) {
				t.Errorf("got error %q, want %q", got, tt.injectErr)
			}
		})
	}
}
