package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"kombiner/pkg/apis/kombiner/v1alpha1"
	"kombiner/pkg/apis/scheduler"
	"kombiner/pkg/generated/clientset/versioned"
)

// DeletePlacementRequestTimeout is the amount of time we wait for deleting a
// placement request we do not care about anymore. This should be enough but
// if it isn't the system is designed to delete them before creating a new
// one for the same pod.
const DeletePlacementRequestTimeout = time.Second

// this global variable is used to ensure, at compile time, that the BindPlugin
// struct complies with the expected framework interface.
var _ framework.BindPlugin = &BindPlugin{}

// BindPlugin implements the framework.BindPlugin interface for binding pods to
// nodes. Its purpose is to generate PlacementRequest for pods and wait until
// they are done.
type BindPlugin struct {
	logger klog.Logger
	client versioned.Interface
	config *scheduler.PlacementRequestBinderArgs
}

// Name purpose is to return the plugin name so the scheduler framework can
// identify it while parsing the configuration.
func (p *BindPlugin) Name() string {
	return PluginName
}

// Bind purpose is to bind the provided pod to the specified, by name, node. It
// should return if the bound succeeded or not so it needs to wait for the
// PlacementRequest to be fulfilled by the placement request controller.
func (p *BindPlugin) Bind(
	ctx context.Context, state *framework.CycleState, pod *corev1.Pod, node string,
) *framework.Status {
	// we will use the pod uid as the placement request name.
	prname := string(pod.UID)
	if prname == "" {
		return framework.AsStatus(fmt.Errorf("pod %v/%v is missing its UID", pod.Namespace, pod.Name))
	}

	schedulerName := pod.Spec.SchedulerName
	if schedulerName == "" {
		schedulerName = corev1.DefaultSchedulerName
	}

	pr := &v1alpha1.PlacementRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prname,
			Namespace: pod.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Pod",
					Name:       pod.Name,
					UID:        pod.UID,
				},
			},
		},
		Spec: v1alpha1.PlacementRequestSpec{
			Policy:        v1alpha1.PlacementRequestPolicyLenient,
			Priority:      0,
			SchedulerName: schedulerName,
			Bindings: []v1alpha1.Binding{
				{
					PodName:  pod.Name,
					PodUID:   pod.UID,
					NodeName: node,
				},
			},
		},
	}

	// XXX here be dragons. the controller does not yet support in place
	// updates (and i am not sure it ever will), due to this we are just
	// deleting if a placement request already exists for the pod. i do
	// expect this to cause issues later down the line.
	client := p.client.KombinerV1alpha1().PlacementRequests(pod.Namespace)
	if _, err := client.Get(ctx, prname, metav1.GetOptions{}); err == nil {
		err := client.Delete(ctx, prname, metav1.DeleteOptions{})
		if err != nil {
			return framework.AsStatus(err)
		}
	}

	if _, err := client.Create(ctx, pr, metav1.CreateOptions{}); err != nil {
		return framework.AsStatus(err)
	}

	// this timeout here is used to limit the amount of time we will spend
	// waiting for the placement request to be resolved.
	timeout, cancel := context.WithTimeout(ctx, p.config.Timeout.Duration)
	defer cancel()

	// XXX we are simply polling here but this is wrong in so many levels.
	// the amount of things to be improved here is huge.
	if err := wait.PollUntilContextCancel(
		timeout, time.Second, true,
		func(ctx context.Context) (bool, error) {
			var err error
			pr, err = client.Get(ctx, prname, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			switch pr.Status.Result {
			case v1alpha1.PlacementRequestResultUnknown:
				return false, nil
			}

			return true, nil
		},
	); err != nil {
		// we don't know what has caused the failure, it may be that
		// the original context was cancelled so we can't use it.
		delctx, delcancel := context.WithTimeout(
			context.Background(), DeletePlacementRequestTimeout,
		)
		defer delcancel()

		err = client.Delete(delctx, prname, metav1.DeleteOptions{})
		if err != nil {
			p.logger.Error(err, "failed to delete placement request")
		}
		return framework.AsStatus(err)
	}

	// in case of failure during the bind process we use the placement
	// request status message as an error string and return. this is to
	// keep backwards compatibility with the default bind plugin
	// implementation.
	if pr.Status.Result != v1alpha1.PlacementRequestResultSuccess {
		return framework.AsStatus(errors.New(pr.Status.Message))
	}

	return framework.NewStatus(framework.Success, pr.Status.Message)
}

// NewBindPlugin creates a new BindPlugin instance. This function is used when
// using this plugin as an extension for the kubernetes scheduler.
func NewBindPlugin(
	ctx context.Context, oargs runtime.Object, handle framework.Handle,
) (framework.Plugin, error) {
	kconfig := handle.KubeConfig()

	args, ok := oargs.(*scheduler.PlacementRequestBinderArgs)
	if !ok {
		return nil, fmt.Errorf("invalid %T type", oargs)
	}

	// make sure we are using sane default values.
	scheduler.SetDefaults(args)

	// XXX here be dragons. it seems like the kubeconfig returned by the
	// framework handle prefers to use protobuf and this is not supported
	// by the current implementation of the placement request types. this
	// "patch" here may affect other plugins using the same kubeconfig but
	// this remains to be seen.
	kconfig.ContentType = "application/json"

	client, err := versioned.NewForConfig(kconfig)
	if err != nil {
		return nil, fmt.Errorf("error building placement request clientset: %w", err)
	}

	// extract the logger from the context and keep it around.
	logger := klog.FromContext(ctx).WithValues("plugin", PluginName)

	return &BindPlugin{
		client: client,
		config: args,
		logger: logger,
	}, nil
}
