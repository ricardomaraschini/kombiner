package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"placement-request-controller/pkg/controller"
	clientset "placement-request-controller/pkg/generated/clientset/versioned"
	informers "placement-request-controller/pkg/generated/informers/externalversions"
)

var (
	KubeConfig string
	Version    = "0.1.0"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger := klog.FromContext(ctx)

	logger.Info("placement request consroller starting", "version", Version)

	config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
	if err != nil {
		logger.Error(err, "error building kubeconfig")
		return
	}

	kubecli, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "error building kubernetes client")
		return
	}

	prcli, err := clientset.NewForConfig(config)
	if err != nil {
		logger.Error(err, "error building kubernetes clientset")
		return
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubecli, time.Second*30)
	prInformerFactory := informers.NewSharedInformerFactory(prcli, time.Second*30)

	controller, err := controller.New(
		ctx,
		prcli,
		kubecli.CoreV1(),
		prInformerFactory.Kombiner().V1alpha1().PlacementRequests(),
		kubeInformerFactory.Core().V1().Pods().Lister(),
	)
	if err != nil {
		logger.Error(err, "error creating controller")
		return
	}

	kubeInformerFactory.Start(ctx.Done())
	prInformerFactory.Start(ctx.Done())

	logger.Info("controller started, waiting for events")
	controller.Run(ctx)
}
