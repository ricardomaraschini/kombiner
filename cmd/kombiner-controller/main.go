package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	configapi "kombiner/pkg/apis/config/v1alpha1"
	kombinerapi "kombiner/pkg/apis/kombiner/v1alpha1"
	kombinerconfig "kombiner/pkg/config"
	"kombiner/pkg/controller"
	clientset "kombiner/pkg/generated/clientset/versioned"
	informers "kombiner/pkg/generated/informers/externalversions"
)

var (
	scheme     = apimachineryruntime.NewScheme()
	KubeConfig string
	Version    = "0.1.0"
	configFile string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kombinerapi.AddToScheme(scheme))
	utilruntime.Must(configapi.AddToScheme(scheme))
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger := klog.FromContext(ctx)

	logger.Info("placement request consroller starting", "version", Version)

	config, err := getConfig(configFile, logger)
	if err != nil {
		logger.Error(err, "Unable to load the configuration")
		return
	}

	kubeConfig := ctrl.GetConfigOrDie()
	if kubeConfig.UserAgent == "" {
		kubeConfig.UserAgent = fmt.Sprintf("kombiner/%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
	}

	kubecli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.Error(err, "error building kubernetes client")
		return
	}

	prcli, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		logger.Error(err, "error building kubernetes clientset")
		return
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubecli, time.Second*30)
	prInformerFactory := informers.NewSharedInformerFactory(prcli, time.Second*30)

	controller, err := controller.New(
		ctx,
		config,
		prcli,
		kubecli.CoreV1(),
		prInformerFactory,
		kubeInformerFactory,
	)
	if err != nil {
		logger.Error(err, "error creating controller")
		return
	}

	kubeInformerFactory.Start(ctx.Done())
	prInformerFactory.Start(ctx.Done())

	if err := controller.WaitForHandlersSync(ctx); err != nil {
		logger.Error(err, "handlers are not fully synchronized")
	}

	logger.Info("controller started, waiting for events")
	controller.Run(ctx)
}

func getConfig(configFile string, logger klog.Logger) (configapi.Configuration, error) {
	config, err := kombinerconfig.Load(scheme, configFile)
	if err != nil {
		return config, err
	}
	configStr, err := kombinerconfig.Encode(scheme, &config)
	if err != nil {
		return config, err
	}
	logger.Info("Successfully loaded configuration", "config", configStr)
	return config, nil
}
