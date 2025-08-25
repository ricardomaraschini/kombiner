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

package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreinformers "k8s.io/client-go/informers"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	internalcache "k8s.io/kubernetes/pkg/scheduler/backend/cache"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/metrics"

	configapi "kombiner/pkg/apis/config/v1alpha1"
	"kombiner/pkg/apis/kombiner/v1alpha1"
	client "kombiner/pkg/generated/clientset/versioned"
	kombinerinformers "kombiner/pkg/generated/informers/externalversions"
	lister "kombiner/pkg/generated/listers/kombiner/v1alpha1"
	helpers "kombiner/pkg/placementrequests/v1alpha1"
	"kombiner/pkg/queue"
)

// PlacementRequestController is a controller for handling PlacementRequests.
type PlacementRequestController struct {
	options

	prlister   lister.PlacementRequestLister
	podlister  corev1listers.PodLister
	client     client.Interface
	coreclient corev1client.CoreV1Interface
	queues     map[string]queue.QueueConfig
	iterator   *queue.QueueIterator

	// registeredHandlers contains the registrations of all handlers. It's used to check if all handlers have finished syncing before the scheduling cycles start.
	registeredHandlers []cache.ResourceEventHandlerRegistration

	cache    internalcache.Cache
	snapshot *internalcache.Snapshot
	profiles Profiles
}

// Run reads PlacementRequsts (already sorted by priority and weigth) and calls
// ScheduleOne for each one of them. This is a blocking function that returns
// only when the provided context is done. XXX some more error handling is
// needed here.
func (controller *PlacementRequestController) Run(ctx context.Context) {
	go controller.iterator.Run(ctx)
	for {
		select {
		case pr := <-controller.iterator.Next:
			if err := controller.ScheduleOne(ctx, pr); err != nil {
				controller.logger.Error(err, "failed to schedule")
			}
		case <-ctx.Done():
			return
		}
	}
}

func runPluginValidation(ctx context.Context, plugin framework.Plugin, pod *v1.Pod, nodeInfo *framework.NodeInfo) error {
	state := framework.NewCycleState()
	preFilterPlugin, ok := plugin.(framework.PreFilterPlugin)
	if ok {
		_, status := preFilterPlugin.PreFilter(ctx, state, pod)
		if !status.IsSuccess() && !status.IsSkip() {
			return status.AsError()
		}
		if status.IsSkip() {
			return nil
		}
	}

	filterPlugin, ok := plugin.(framework.FilterPlugin)
	if !ok {
		return fmt.Errorf("validation plugin missing Filter extension point")
	}

	status := filterPlugin.Filter(ctx, state, pod, nodeInfo)
	if !status.IsSuccess() {
		return status.AsError()
	}
	return nil
}

// ScheduleOne is the function responsible for evaluating if a PlacementRequest
// is valid and then bind it to the nodes. This function also sets the status
// once it is finished.
func (controller *PlacementRequestController) ScheduleOne(ctx context.Context, pr *v1alpha1.PlacementRequest) error {
	prid := map[string]string{"name": pr.Name, "namespace": pr.Namespace}
	controller.logger.V(3).Info("processing placement request", "obj", prid)

	// if the placement request is deleted or if its status is known
	// (failure or success), we do not need to process it anymore.
	if pr.DeletionTimestamp != nil || pr.Status.Result != v1alpha1.PlacementRequestResultUnknown {
		controller.logger.V(3).Info("skipping placement request", "obj", prid)
		return nil
	}

	// here we create a few shortcuts to api access entities we are going
	// to use during this function. these shortcuts are already namespace
	// scoped.
	prqclient := controller.client.KombinerV1alpha1().PlacementRequests(pr.Namespace)
	podlister := controller.podlister.Pods(pr.Namespace)

	if err := helpers.Validate(pr); err != nil {
		controller.logger.Error(err, "placement request is not valid", "obj", prid)
		pr.Status.Result = v1alpha1.PlacementRequestResultRejected
		pr.Status.Message = err.Error()
		_, err := prqclient.UpdateStatus(ctx, pr, metav1.UpdateOptions{})
		return err
	}

	// run validing plugins
	if err := controller.cache.UpdateSnapshot(klog.FromContext(ctx), controller.snapshot); err != nil {
		controller.logger.Error(err, "unable to update a snapshot")
		return err
	}

	nodeInfosList, err := controller.snapshot.NodeInfos().List()
	if err != nil {
		controller.logger.Error(err, "unable to get node infos list")
		return err
	}

	nodeInfosMap := make(map[string]*framework.NodeInfo)
	for _, nodeInfo := range nodeInfosList {
		nodeInfosMap[nodeInfo.Node().Name] = nodeInfo
	}

	profile, ok := controller.profiles[pr.Spec.SchedulerName]
	if !ok {
		panic(fmt.Errorf("this should not happen, missing profile for %q scheduler", pr.Spec.SchedulerName))
	}

	for _, binding := range pr.Spec.Bindings {
		controller.logger.V(3).Info("attemping to bind pod to node", "bind", binding, "obj", prid)

		validationFailed := false
		if pod, err := podlister.Get(binding.PodName); err != nil {
			controller.logger.Error(err, "failed to get pod")
			message := fmt.Sprintf("Failed to get pod %s: %v", binding.PodName, err)
			helpers.SetPodBindingFailure(pr, binding, "API error", message)
			continue
		} else if pod.Spec.NodeName != "" {
			if pod.Spec.NodeName == binding.NodeName {
				helpers.SetPodBindingSuccess(pr, binding, "Binding unneeded", "Pod was already bound")
				continue
			}
			message := fmt.Sprintf("Pod %s bound to a different node node", binding.PodName)
			helpers.SetPodBindingFailure(pr, binding, "Pod already bound", message)
			continue
		} else {
			// The plugins can be run in any order as all have to pass
			// Later, these can be run in parallel.
			for pluginName, plugin := range profile.plugins {
				nodeInfo, exists := nodeInfosMap[binding.NodeName]
				if !exists {
					err := fmt.Errorf("nodeinfo missing for a node")
					controller.logger.Error(err, "validation failed", "plugin", pluginName, "pod", klog.KObj(pod))
					helpers.SetPodBindingFailure(pr, binding, "validation failed", err.Error())
					validationFailed = true
					break
				}
				if err := runPluginValidation(ctx, plugin, pod, nodeInfo); err != nil {
					controller.logger.Error(err, "validation failed", "plugin", pluginName, "pod", klog.KObj(pod))
					helpers.SetPodBindingFailure(pr, binding, "validation failed", err.Error())
					validationFailed = true
					break
				}
				controller.logger.V(4).Info("validation passing", "plugin", pluginName, "obj", prid)
			}
		}

		// Break the binding loop since the validation failed.
		// There's no point of running binding/validation of the remaining pods.
		if validationFailed {
			break
		}

		bind := &v1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pr.Namespace,
				Name:      binding.PodName,
				UID:       binding.PodUID,
			},
			Target: v1.ObjectReference{
				Kind: "Node",
				Name: binding.NodeName,
			},
		}

		binder := controller.coreclient.Pods(pr.Namespace)
		if err := binder.Bind(ctx, bind, metav1.CreateOptions{}); err != nil {
			controller.logger.Error(err, "failed to bind pod to node", "bind", binding, "obj", prid)
			helpers.SetPodBindingFailure(pr, binding, "API denied binding", err.Error())
			continue
		}

		controller.logger.V(3).Info("pod successfully bound to node", "bind", binding, "obj", prid)
		helpers.SetPodBindingSuccess(pr, binding, "Binding successful", "Pod successfully bound")
		// TODO(ingvagabund): assume the pod + wire the binding to the cache
	}

	pr.Status.Result, pr.Status.Message = helpers.AssessResult(pr)
	if _, err := prqclient.UpdateStatus(ctx, pr, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update placement request status: %w", err)
	}

	controller.logger.V(3).Info("placement request processed", "obj", prid)
	return nil
}

// enqueue is called when a PlacementRequest is created on the cluster. This
// function responsibility is to enqueue the respective PlacementRequest object
// into one of our internal queues. We have one internal queue per scheduler
// name. If a queue for the scheduler name does not exist, we create it
// automatically. We should not take much long here as we haven't not yet
// enqueued the placement request and there may be more events happening. We
// do some basic validation here and in case of failure we just try to
// reject the PlacementRequest.
func (controller *PlacementRequestController) enqueue(obj interface{}) {
	pr, ok := obj.(*v1alpha1.PlacementRequest)
	if !ok || pr.Spec.SchedulerName == "" {
		return
	}

	qcfg, found := controller.queues[pr.Spec.SchedulerName]
	if !found {
		reason, msg := "QueueNotFound", "Scheduler queue not found"
		controller.TryToRejectPlacementRequest(pr, reason, msg)
		return
	}

	if len(pr.Spec.Bindings) > int(qcfg.MaxSize) {
		reason, msg := "PlacementRequestTooLarge", "Placement request too large"
		controller.TryToRejectPlacementRequest(pr, reason, msg)
		return
	}

	qcfg.QueueRef.Push(pr)
}

// TryToRejectPlacementRequest should be used when rejecting a PlacementRequest
// without worrying about possible failures when doing so. This function uses a
// hard coded timeout and does not return (but logs) errors.
func (controller *PlacementRequestController) TryToRejectPlacementRequest(
	pr *v1alpha1.PlacementRequest, reason, message string,
) {
	prid := map[string]string{"name": pr.Name, "namespace": pr.Namespace}
	controller.logger.V(5).Info("trying to reject placement request", "obj", prid)

	ctx, cancel := context.WithTimeout(
		context.Background(), controller.options.tryToRejectTimeout,
	)
	defer cancel()

	pr.Status.Result = v1alpha1.PlacementRequestResultRejected
	pr.Status.Reason = reason
	pr.Status.Message = message

	prqclient := controller.client.KombinerV1alpha1().PlacementRequests(pr.Namespace)
	if _, err := prqclient.UpdateStatus(ctx, pr, metav1.UpdateOptions{}); err != nil {
		controller.logger.Error(err, "failed to reject placement request", "obj", prid)
	}
}

// New returns a PlacementRequest controller.
func New(
	ctx context.Context,
	cfg configapi.Configuration,
	client client.Interface,
	coreclient corev1client.CoreV1Interface,
	kombinerInformerFactory kombinerinformers.SharedInformerFactory,
	coreinformerFactory coreinformers.SharedInformerFactory,
	opts ...Option,
) (*PlacementRequestController, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	configs := queue.QueueConfigFromV1Alpha1Config(cfg)
	if err := configs.Validate(); err != nil {
		return nil, fmt.Errorf("invalid queue configuration: %w", err)
	}

	itopts := []queue.QueueIteratorOption{}
	switch cfg.FairnessAlgorithm {
	case "", configapi.RoundRobin:
		options.logger.Info("using the default round-robin fairness algorithm")
		itopts = append(itopts, queue.WithReaderFactory(queue.NewRoundRobinReader))
	case configapi.Uniform:
		options.logger.Info("using the uniform fairness algorithm")
		itopts = append(itopts, queue.WithReaderFactory(queue.NewUniformReader))
	default:
		return nil, fmt.Errorf("unknown fairness algorithm %q", cfg.FairnessAlgorithm)
	}

	iterator, err := queue.NewQueueIterator(configs, itopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create internal queue iterator: %w", err)
	}

	// instantiate the queue profiles
	snapshot := internalcache.NewEmptySnapshot()

	metrics.Register()

	controller := &PlacementRequestController{
		options:    options,
		client:     client,
		coreclient: coreclient,
		podlister:  coreinformerFactory.Core().V1().Pods().Lister(),
		prlister:   kombinerInformerFactory.Kombiner().V1alpha1().PlacementRequests().Lister(),
		queues:     configs.ToMap(),
		iterator:   iterator,
		cache:      internalcache.New(ctx, 0),
		snapshot:   snapshot,
	}

	if err := controller.addEventHandlers(coreinformerFactory, kombinerInformerFactory); err != nil {
		return nil, fmt.Errorf("failed to add event handlers: %w", err)
	}

	controller.profiles = controller.pluginProfilesFromV1Alpha1Config(
		ctx,
		cfg,
		snapshot,
		coreinformerFactory,
	)

	return controller, nil
}
