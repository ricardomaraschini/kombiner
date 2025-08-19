package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	configapi "kombiner/pkg/apis/config/v1alpha1"
	"kombiner/pkg/apis/kombiner/v1alpha1"
	client "kombiner/pkg/generated/clientset/versioned"
	informer "kombiner/pkg/generated/informers/externalversions/kombiner/v1alpha1"
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

	// here we create a few shortcuts to api access entities we are going
	// to use during this function. these shortcuts are already namespace
	// scoped.
	prqclient := prc.client.KombinerV1alpha1().PlacementRequests(pr.Namespace)
	podlister := prc.podlister.Pods(pr.Namespace)

	if err := helpers.Validate(pr); err != nil {
		prc.logger.Error(err, "placement request is not valid", "obj", prid)
		pr.Status.Result = v1alpha1.PlacementRequestResultRejected
		pr.Status.Message = err.Error()
		_, err := prqclient.UpdateStatus(ctx, pr, metav1.UpdateOptions{})
		return err
	}

	for _, binding := range pr.Spec.Bindings {
		prc.logger.V(3).Info("binding pod to node", "bind", binding, "obj", prid)

		if pod, err := podlister.Get(binding.PodName); err != nil {
			prc.logger.Error(err, "failed to get pod")
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

		binder := prc.coreclient.Pods(pr.Namespace)
		if err := binder.Bind(ctx, bind, metav1.CreateOptions{}); err != nil {
			prc.logger.Error(err, "failed to bind pod to node", "bind", binding, "obj", prid)
			helpers.SetPodBindingFailure(pr, binding, "API denied binding", err.Error())
			continue
		}

		prc.logger.V(3).Info("pod successfully bound to node", "bind", binding, "obj", prid)
		helpers.SetPodBindingSuccess(pr, binding, "Binding successful", "Pod successfully bound")
	}

	pr.Status.Result, pr.Status.Message = helpers.AssessResult(pr)
	if _, err := prqclient.UpdateStatus(ctx, pr, metav1.UpdateOptions{}); err != nil {
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

	// We only process placement requests for known schedulers.
	queue, ok := prc.queues[pr.Spec.SchedulerName]
	if !ok {
		err := fmt.Errorf("missing queue %q configuration", pr.Spec.SchedulerName)
		prc.logger.Error(err, "unable to enqueue placement request")
		return
	}

	queue.Push(pr)
}

// New returns a PlacementRequest controller.
func New(
	ctx context.Context,
	cfg configapi.Configuration,
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

	controller := &PlacementRequestController{
		options:    options,
		client:     client,
		coreclient: coreclient,
		podlister:  podlister,
		prlister:   informer.Lister(),
		queues:     configs.ToMap(),
		iterator:   iterator,
	}

	if err := controller.AddEventHandlers(informer); err != nil {
		return nil, fmt.Errorf("failed to add event handlers: %w", err)
	}

	return controller, nil
}
