package e2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"kombiner/pkg/apis/kombiner/v1alpha1"
	"kombiner/pkg/generated/clientset/versioned"
)

var _ = ginkgo.Describe("create a oversized placement request", func() {
	var prclientset *versioned.Clientset

	ginkgo.BeforeEach(
		func() {
			config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			prclientset, err = versioned.NewForConfig(config)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		},
	)

	testName := "oversized-pr"
	testNamespace := "default"
	ginkgo.It("should reject the request with proper error",
		func() {
			pr := &v1alpha1.PlacementRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: v1alpha1.PlacementRequestSpec{
					Policy:        v1alpha1.PlacementRequestPolicyLenient,
					SchedulerName: "kombiner-scheduler",
					Bindings: []v1alpha1.Binding{
						{
							NodeName: "node1",
							PodName:  "pod1",
							PodUID:   "pod1-uid",
						},
						{
							NodeName: "node2",
							PodName:  "pod2",
							PodUID:   "pod2-uid",
						},
					},
				},
			}

			cli := prclientset.KombinerV1alpha1().PlacementRequests(testNamespace)
			_, err := cli.Create(context.TODO(), pr, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(
				func() (v1alpha1.PlacementRequestResult, error) {
					pr, err := cli.Get(
						context.TODO(), testName, metav1.GetOptions{},
					)
					if err != nil {
						return "", err
					}
					return pr.Status.Result, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal(v1alpha1.PlacementRequestResultRejected))

			gomega.Eventually(
				func() (string, error) {
					pr, err := cli.Get(
						context.TODO(), testName, metav1.GetOptions{},
					)
					if err != nil {
						return "", err
					}
					return pr.Status.Reason, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal("PlacementRequestTooLarge"))

			err = cli.Delete(context.TODO(), testName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		},
	)
})

var _ = ginkgo.Describe("create a placement request for an unknown scheduler", func() {
	var prclientset *versioned.Clientset

	ginkgo.BeforeEach(
		func() {
			config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			prclientset, err = versioned.NewForConfig(config)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		},
	)

	testName := "oversized-pr"
	testNamespace := "default"
	ginkgo.It("should reject the request with proper error",
		func() {
			pr := &v1alpha1.PlacementRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: v1alpha1.PlacementRequestSpec{
					Policy:        v1alpha1.PlacementRequestPolicyLenient,
					SchedulerName: "unknown-scheduler",
					Bindings: []v1alpha1.Binding{
						{
							NodeName: "node1",
							PodName:  "pod1",
							PodUID:   "pod1-uid",
						},
					},
				},
			}

			cli := prclientset.KombinerV1alpha1().PlacementRequests(testNamespace)
			_, err := cli.Create(context.TODO(), pr, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(
				func() (v1alpha1.PlacementRequestResult, error) {
					pr, err := cli.Get(
						context.TODO(), testName, metav1.GetOptions{},
					)
					if err != nil {
						return "", err
					}
					return pr.Status.Result, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal(v1alpha1.PlacementRequestResultRejected))

			gomega.Eventually(
				func() (string, error) {
					pr, err := cli.Get(
						context.TODO(), testName, metav1.GetOptions{},
					)
					if err != nil {
						return "", err
					}
					return pr.Status.Reason, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal("QueueNotFound"))

			err = cli.Delete(context.TODO(), testName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		},
	)
})
