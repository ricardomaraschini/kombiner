package e2e

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"kombiner/pkg/apis/kombiner/v1alpha1"
	"kombiner/pkg/generated/clientset/versioned"
)

var _ = ginkgo.Describe("create a single pod", func() {
	var clientset *kubernetes.Clientset
	var prclientset *versioned.Clientset

	ginkgo.BeforeEach(
		func() {
			config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clientset, err = kubernetes.NewForConfig(config)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			prclientset, err = versioned.NewForConfig(config)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		},
	)

	ginkgo.It("should create a placement request",
		func() {
			testNamespace, testPod := "default", "demo-pod"
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: testPod},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					SchedulerName: "kombiner-scheduler",
					Containers: []corev1.Container{
						{
							Name:    "container",
							Image:   "fedora:latest",
							Command: []string{"sleep"},
							Args:    []string{"10"},
						},
					},
				},
			}

			_, err := clientset.CoreV1().Pods(testNamespace).Create(
				context.TODO(), pod, metav1.CreateOptions{},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(
				func() (corev1.PodPhase, error) {
					pod, err = clientset.
						CoreV1().
						Pods(testNamespace).
						Get(
							context.TODO(),
							testPod,
							metav1.GetOptions{},
						)
					if err != nil {
						return "", err
					}
					return pod.Status.Phase, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal(corev1.PodRunning))

			gomega.Eventually(
				func() (v1alpha1.PlacementRequestResult, error) {
					pr, err := prclientset.
						KombinerV1alpha1().
						PlacementRequests(testNamespace).
						Get(
							context.TODO(),
							string(pod.UID),
							metav1.GetOptions{},
						)
					if err != nil {
						return "", err
					}
					return pr.Status.Result, nil
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.Equal(v1alpha1.PlacementRequestResultSuccess))

			err = clientset.CoreV1().Pods(testNamespace).Delete(
				context.TODO(), testPod, metav1.DeleteOptions{},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(
				func() bool {
					_, err := prclientset.
						KombinerV1alpha1().
						PlacementRequests(testNamespace).
						Get(
							context.TODO(),
							string(pod.UID),
							metav1.GetOptions{},
						)
					return errors.IsNotFound(err)
				},
				30*time.Second, 2*time.Second,
			).Should(gomega.BeTrue())
		},
	)
})
