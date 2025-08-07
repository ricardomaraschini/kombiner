package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"placement-request-controller/pkg/apis/v1alpha1"
	client "placement-request-controller/pkg/generated/clientset/versioned"
	informer "placement-request-controller/pkg/generated/informers/externalversions/apis/v1alpha1"
	lister "placement-request-controller/pkg/generated/listers/apis/v1alpha1"
	"placement-request-controller/pkg/queue"
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
			if err := prc.ScheduleOne(ctx, pr); err != nil {
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
func (prc *PlacementRequestController) ScheduleOne(ctx context.Context, pr *v1alpha1.PlacementRequest) error {
	prid := map[string]string{"name": pr.Name, "namespace": pr.Namespace}
	prc.logger.V(3).Info("processing placement request", "obj", prid)

	// if the placement request is deleted or if its status is known
	// (failure or success), we do not need to process it anymore.
	if pr.DeletionTimestamp != nil || pr.Status.Result != v1alpha1.PlacementRequestResultUnknown {
		prc.logger.V(3).Info("skipping placement request", "obj", prid)
		return nil
	}

	// if there is no binding present then we cannot process the
	// PlacementRequest. XXX this should be handled at the api level.
	if len(pr.Spec.Bindings) == 0 {
		prc.logger.V(3).Info("no bindings found", "obj", prid)
		pr.Status.Result = v1alpha1.PlacementRequestResultRejected
		pr.Status.Message = "The request was rejected because it has no bindings"
		updater := prc.client.SchedulingV1alpha1().PlacementRequests(pr.Namespace)
		_, err := updater.Update(ctx, pr, metav1.UpdateOptions{})
		return err
	}

	// we only operate on Lenient policy. for the all or nothing policy a
	// change in the api server will be necessary.
	if pr.Spec.Policy != v1alpha1.PlacementRequestPolicyLenient {
		prc.logger.V(3).Info("unsupported policy", "policy", pr.Spec.Policy, "obj", prid)
		pr.Status.Result = v1alpha1.PlacementRequestResultRejected
		pr.Status.Message = fmt.Sprintf("Unsupported policy: %s", pr.Spec.Policy)
		updater := prc.client.SchedulingV1alpha1().PlacementRequests(pr.Namespace)
		_, err := updater.Update(ctx, pr, metav1.UpdateOptions{})
		return err
	}

	var success bool
	for _, binding := range pr.Spec.Bindings {
		prc.logger.V(3).Info("binding pod to node", "bind", binding, "obj", prid)

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

		binder := prc.coreclient.Pods(pr.Namespace)
		if err := binder.Bind(ctx, bind, metav1.CreateOptions{}); err != nil {
			prc.logger.Error(err, "failed to bind pod to node", "bind", binding, "obj", prid)
			pr.Status.Bindings = append(
				pr.Status.Bindings,
				v1alpha1.PlacementRequestBindingResult{
					Binding: binding,
					Result:  v1alpha1.PlacementRequestResultFailure,
					Reason:  "API denied binding",
					Message: err.Error(),
				},
			)
			continue
		}

		success = true
		prc.logger.V(3).Info("pod successfully bound to node", "bind", binding, "obj", prid)
		pr.Status.Bindings = append(
			pr.Status.Bindings,
			v1alpha1.PlacementRequestBindingResult{
				Binding: binding,
				Result:  v1alpha1.PlacementRequestResultSuccess,
				Reason:  "Binding successful",
				Message: "The pod was successfully bound to the node",
			},
		)
	}

	pr.Status.Result = v1alpha1.PlacementRequestResultSuccess
	pr.Status.Message = "The request was successfully scheduled"
	if !success {
		pr.Status.Result = v1alpha1.PlacementRequestResultFailure
		pr.Status.Message = "All bindings failed"
	}

	updater := prc.client.SchedulingV1alpha1().PlacementRequests(pr.Namespace)
	if _, err := updater.UpdateStatus(ctx, pr, metav1.UpdateOptions{}); err != nil {
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
