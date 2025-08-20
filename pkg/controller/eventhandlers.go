package controller

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"kombiner/pkg/apis/kombiner/v1alpha1"
	kombinerinformers "kombiner/pkg/generated/informers/externalversions"
)

const (
	// syncedPollPeriod controls how often you look at the status of your sync funcs
	syncedPollPeriod = 100 * time.Millisecond
)

func (controller *PlacementRequestController) addPodToCache(obj interface{}) {
	logger := controller.logger
	pod, ok := obj.(*v1.Pod)
	if !ok {
		logger.Error(nil, "Cannot convert to *v1.Pod", "obj", obj)
		return
	}

	logger.V(3).Info("Add event for a pod", "pod", klog.KObj(pod))
	if err := controller.cache.AddPod(logger, pod); err != nil {
		logger.Error(err, "Kombiner cache addPod failed", "pod", klog.KObj(pod))
	}
}

func (controller *PlacementRequestController) updatePodInCache(oldObj, newObj interface{}) {
	logger := controller.logger
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		logger.Error(nil, "Cannot convert oldObj to *v1.Pod", "oldObj", oldObj)
		return
	}
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		logger.Error(nil, "Cannot convert newObj to *v1.Pod", "newObj", newObj)
		return
	}

	logger.V(4).Info("Update event for a pod", "pod", klog.KObj(oldPod))
	if err := controller.cache.UpdatePod(logger, oldPod, newPod); err != nil {
		logger.Error(err, "Kombiner cache updatePod failed", "pod", klog.KObj(oldPod))
	}
}

func (controller *PlacementRequestController) deletePodFromCache(obj interface{}) {
	logger := controller.logger
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			logger.Error(nil, "Cannot convert to *v1.Pod", "obj", t.Obj)
			return
		}
	default:
		logger.Error(nil, "Cannot convert to *v1.Pod", "obj", t)
		return
	}

	logger.V(3).Info("Delete event for a pod", "pod", klog.KObj(pod))
	if err := controller.cache.RemovePod(logger, pod); err != nil {
		logger.Error(err, "Kombiner cache removePod failed", "pod", klog.KObj(pod))
	}
}

func (controller *PlacementRequestController) addNodeToCache(obj interface{}) {
	logger := controller.logger
	node, ok := obj.(*v1.Node)
	if !ok {
		logger.Error(nil, "Cannot convert to *v1.Node", "obj", obj)
		return
	}

	logger.V(3).Info("Add event for node", "node", klog.KObj(node))
	controller.cache.AddNode(logger, node)
}

func (controller *PlacementRequestController) updateNodeInCache(oldObj, newObj interface{}) {
	logger := controller.logger
	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		logger.Error(nil, "Cannot convert oldObj to *v1.Node", "oldObj", oldObj)
		return
	}
	newNode, ok := newObj.(*v1.Node)
	if !ok {
		logger.Error(nil, "Cannot convert newObj to *v1.Node", "newObj", newObj)
		return
	}

	logger.V(4).Info("Update event for node", "node", klog.KObj(newNode))
	controller.cache.UpdateNode(logger, oldNode, newNode)
}

func (controller *PlacementRequestController) deleteNodeFromCache(obj interface{}) {
	logger := controller.logger
	var node *v1.Node
	switch t := obj.(type) {
	case *v1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*v1.Node)
		if !ok {
			logger.Error(nil, "Cannot convert to *v1.Node", "obj", t.Obj)
			return
		}
	default:
		logger.Error(nil, "Cannot convert to *v1.Node", "obj", t)
		return
	}

	logger.V(3).Info("Delete event for node", "node", klog.KObj(node))
	if err := controller.cache.RemoveNode(logger, node); err != nil {
		logger.Error(err, "Kombiner cache RemoveNode failed")
	}
}

// addEventHandlers is used to make sure the informers are pointing to the
// right event handlers here. We want to enqueue every new PlacementRequest
// into our internal queues.
func (controller *PlacementRequestController) addEventHandlers(
	coreinformerFactory coreinformers.SharedInformerFactory,
	kombinerInformerFactory kombinerinformers.SharedInformerFactory,
) error {
	var (
		handlerRegistration cache.ResourceEventHandlerRegistration
		err                 error
		handlers            []cache.ResourceEventHandlerRegistration
	)

	if handlerRegistration, err = kombinerInformerFactory.Kombiner().V1alpha1().PlacementRequests().Informer().AddEventHandler(
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
				AddFunc: controller.enqueue,
			},
		},
	); err != nil {
		return fmt.Errorf("failed to add placement request event handler: %w", err)
	}
	handlers = append(handlers, handlerRegistration)

	// scheduled pod cache
	if handlerRegistration, err = coreinformerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return true
				case cache.DeletedFinalStateUnknown:
					if _, ok := t.Obj.(*v1.Pod); ok {
						// The carried object may be stale, so we don't use it to check if
						// it's assigned or not. Attempting to cleanup anyways.
						return true
					}
					utilruntime.HandleError(fmt.Errorf("unable to convert object %T to *v1.Pod", obj))
					return false
				default:
					utilruntime.HandleError(fmt.Errorf("unable to handle object: %T", obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    controller.addPodToCache,
				UpdateFunc: controller.updatePodInCache,
				DeleteFunc: controller.deletePodFromCache,
			},
		},
	); err != nil {
		return err
	}
	handlers = append(handlers, handlerRegistration)

	if handlerRegistration, err = coreinformerFactory.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addNodeToCache,
			UpdateFunc: controller.updateNodeInCache,
			DeleteFunc: controller.deleteNodeFromCache,
		},
	); err != nil {
		return err
	}
	handlers = append(handlers, handlerRegistration)

	controller.registeredHandlers = handlers
	return nil
}

func (controller *PlacementRequestController) WaitForHandlersSync(ctx context.Context) error {
	return wait.PollUntilContextCancel(ctx, syncedPollPeriod, true, func(ctx context.Context) (done bool, err error) {
		for _, handler := range controller.registeredHandlers {
			if !handler.HasSynced() {
				return false, nil
			}
		}
		return true, nil
	})
}
