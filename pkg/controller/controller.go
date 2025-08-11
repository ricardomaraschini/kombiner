package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"kombiner/pkg/apis/v1alpha1"
	"kombiner/pkg/composition"
	client "kombiner/pkg/generated/clientset/versioned"
	informer "kombiner/pkg/generated/informers/externalversions/apis/v1alpha1"
	lister "kombiner/pkg/generated/listers/apis/v1alpha1"
	"kombiner/pkg/queue"
)

// PlacementRequestController is a controller for handling PlacementRequests.
type PlacementRequestController struct {
	options

	prlister   lister.PlacementRequestLister
	podlister  corev1listers.PodLister
	client     client.Interface
	coreclient corev1client.CoreV1Interface
	queues     map[string]*queue.PlacementRequestQueue
	iterator   *queue.QueueIterator
}

// Run reads PlacementRequsts (already sorted by priority and weigth) and calls
// ScheduleOne for each one of them. This is a blocking function that returns
// only when the provided context is done. XXX some more error handling is
// needed here.
func (prc *PlacementRequestController) Run(ctx context.Context) {
	go prc.iterator.Run(ctx)
	for {
		select {
		case pr := <-prc.iterator.Next:
			composite := &composition.PlacementRequest{PlacementRequest: pr}
			if err := prc.ScheduleOne(ctx, composite); err != nil {
				prc.logger.Error(err, "failed to schedule")
			}
		case <-ctx.Done():
			return
		}
	}
}

// ScheduleOne is the function responsible for evaluating if a PlacementRequest
// is valid and then bind it to the nodes. This function also sets the status
// once it is finished.
func (prc *PlacementRequestController) ScheduleOne(ctx context.Context, pr *composition.PlacementRequest) error {
	prid := map[string]string{"name": pr.Name, "namespace": pr.Namespace}
	prc.logger.V(3).Info("processing placement request", "obj", prid)

	prcli := prc.client.KombinerV1alpha1().PlacementRequests(pr.Namespace)
	podcli := prc.coreclient.Pods(pr.Namespace)
	podlister := prc.podlister.Pods(pr.Namespace)

	// if the placement request is deleted or if its status is known
	// (failure or success), we do not need to process it anymore.
	if pr.DeletionTimestamp != nil || pr.Status.Result != v1alpha1.PlacementRequestResultUnknown {
		prc.logger.V(3).Info("skipping placement request", "obj", prid)
		return nil
	}

	// validate ensures that the placement request is valid. Most of the
	// validations are executed through the open api definition but if that
	// is beyond our control so we want to ensure we only process if valid.
	if !pr.Valid() {
		prc.logger.V(3).Info("placement request is invalid, skipping", "obj", prid)
		_, err := prcli.Update(ctx, pr.PlacementRequest, metav1.UpdateOptions{})
		return err
	}

	for _, binding := range pr.Spec.Bindings {
		prc.logger.V(3).Info("binding pod to node", "bind", binding, "obj", prid)

		if pod, err := podlister.Get(binding.PodName); err != nil {
			pr.SetBindingFailure(binding, "API error", err.Error())
			continue
		} else if pod.Spec.NodeName != "" {
			pr.SetBindingFailure(binding, "Pod already bound", "Already scheduled pod")
			continue
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

		if err := podcli.Bind(ctx, bind, metav1.CreateOptions{}); err != nil {
			prc.logger.Error(err, "failed to bind pod to node", "bind", binding, "obj", prid)
			pr.SetBindingFailure(binding, "API denied binding", err.Error())
			continue
		}

		prc.logger.V(3).Info("pod successfully bound to node", "bind", binding, "obj", prid)
		pr.SetBindingSuccess(binding)
	}

	pr.AssessResult()

	_, err := prcli.UpdateStatus(ctx, pr.PlacementRequest, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update placement request status: %w", err)
	}

	prc.logger.V(3).Info("placement request processed", "obj", prid)
	return nil
}

// AddEventHandlers is used to make sure the informers are pointing to the
// right event handlers here. We want to enqueue every new PlacementRequest
// into our internal queues.
func (prc *PlacementRequestController) AddEventHandlers(informer informer.PlacementRequestInformer) error {
	if _, err := informer.Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch obj.(type) {
				case *v1alpha1.PlacementRequest:
					return true
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: prc.enqueue,
			},
		},
	); err != nil {
		return fmt.Errorf("failed to add placement request event handler: %w", err)
	}
	return nil
}

// enqueue is called when a PlacementRequest is created on the cluster. This
// function responsibility is to enqueue the respective PlacementRequest object
// into one of our internal queues. We have one internal queue per scheduler
// name. If a queue for the scheduler name does not exist, we create it
// automatically.
func (prc *PlacementRequestController) enqueue(obj interface{}) {
	pr, ok := obj.(*v1alpha1.PlacementRequest)
	if !ok || pr.Spec.SchedulerName == "" {
		return
	}

	// if we already have a queue for the scheduler name, we just push
	// the PlacementRequest into it. they are going to be sorted by their
	// priority.
	if queue, ok := prc.queues[pr.Spec.SchedulerName]; ok {
		queue.Push(pr)
		return
	}

	// at this point we do not have a queue for the scheduler name, so we
	// need to create one and enqueue the PlacementRequest. XXX Weight here
	// should be read from the configuration file. Also, some more logging
	// is needed here.
	config := queue.QueueConfig{
		Name:   pr.Spec.SchedulerName,
		Weight: 1,
		Queue:  queue.NewPlacementRequestQueue(),
	}

	if err := prc.iterator.AddQueue(config); err != nil {
		return
	}

	prc.queues[pr.Spec.SchedulerName] = config.Queue
	config.Queue.Push(pr)
}

// New returns a PlacementRequest controller.
func New(
	ctx context.Context,
	client client.Interface,
	coreclient corev1client.CoreV1Interface,
	informer informer.PlacementRequestInformer,
	podlister corev1listers.PodLister,
	opts ...Option,
) (*PlacementRequestController, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	iterator, err := queue.NewQueueIterator()
	if err != nil {
		return nil, fmt.Errorf("failed to create internal queue iterator: %w", err)
	}

	controller := &PlacementRequestController{
		options:    options,
		client:     client,
		coreclient: coreclient,
		podlister:  podlister,
		prlister:   informer.Lister(),
		queues:     map[string]*queue.PlacementRequestQueue{},
		iterator:   iterator,
	}

	if err := controller.AddEventHandlers(informer); err != nil {
		return nil, fmt.Errorf("failed to add event handlers: %w", err)
	}

	return controller, nil
}
