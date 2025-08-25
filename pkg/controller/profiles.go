package controller

import (
	"context"
	"fmt"

	coreinformers "k8s.io/client-go/informers"
	schedulerconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
	internalcache "k8s.io/kubernetes/pkg/scheduler/backend/cache"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	schedulingpluginnames "k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"

	configapi "kombiner/pkg/apis/config/v1alpha1"
)

type QueueProfile struct {
	frmwrk  framework.Framework
	plugins map[string]framework.Plugin
}

type Profiles map[string]*QueueProfile

func (controller *PlacementRequestController) pluginProfilesFromV1Alpha1Config(
	ctx context.Context,
	raw configapi.Configuration,
	snapshot *internalcache.Snapshot,
	coreinformerFactory coreinformers.SharedInformerFactory,
) Profiles {
	profiles := make(Profiles)

	for _, config := range raw.Queues {
		profile := &QueueProfile{
			plugins: make(map[string]framework.Plugin),
		}

		frmwrk, err := frameworkruntime.NewFramework(
			ctx,
			nil,
			nil,
			frameworkruntime.WithSnapshotSharedLister(snapshot),
			frameworkruntime.WithInformerFactory(coreinformerFactory),
		)
		if err != nil {
			panic(fmt.Errorf("unable to create a framework: %v", err))
		}
		profile.frmwrk = frmwrk

		// INFO(ingvagabund): temporary solution for enabling certain plugins
		for _, plugin := range config.Plugins.Validate.Enabled {
			switch plugin {
			case schedulingpluginnames.TaintToleration:
				// context, args, framework handle, featuregates
				p, err := tainttoleration.New(ctx, nil, frmwrk, feature.Features{})
				if err != nil {
					panic(fmt.Errorf("unable to create a plugin: %v", err))
				}
				profile.plugins[plugin] = p
				controller.logger.V(3).Info("plugin initialized", "plugin", plugin)
			case schedulingpluginnames.PodTopologySpread:
				// context, args, framework handle, featuregates
				// INFO(ingvagabund): temporary inject default args
				p, err := podtopologyspread.New(ctx, &schedulerconfig.PodTopologySpreadArgs{
					DefaultingType: schedulerconfig.SystemDefaulting,
				}, frmwrk, feature.Features{})
				if err != nil {
					panic(fmt.Errorf("unable to create a plugin: %v", err))
				}
				profile.plugins[plugin] = p
				controller.logger.V(3).Info("plugin initialized", "plugin", plugin)
			case schedulingpluginnames.NodeAffinity:
				// context, args, framework handle, featuregates
				// INFO(ingvagabund): temporary inject default args
				p, err := nodeaffinity.New(ctx, &schedulerconfig.NodeAffinityArgs{}, frmwrk, feature.Features{})
				if err != nil {
					panic(fmt.Errorf("unable to create a plugin: %v", err))
				}
				profile.plugins[plugin] = p
				controller.logger.V(3).Info("plugin initialized", "plugin", plugin)
			}
		}
		profiles[config.SchedulerName] = profile
	}

	return profiles
}
