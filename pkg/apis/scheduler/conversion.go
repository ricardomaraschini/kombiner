package scheduler

import (
	"fmt"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
	v1 "k8s.io/kubernetes/pkg/scheduler/apis/config/v1"

	"kombiner/pkg/apis/scheduler/v1alpha1"
)

// RegisterConversions registers in the provided Scheme the conversion
// functions for the external and internal api types.
func RegisterConversions(scheme *runtime.Scheme) error {
	if err := scheme.AddConversionFunc(
		(*v1alpha1.PlacementRequestBinderArgs)(nil),
		(*PlacementRequestBinderArgs)(nil),
		v1alpha1ToInternal,
	); err != nil {
		return fmt.Errorf("adding v1alpha1 to internal conversion: %v", err)
	}

	if err := scheme.AddConversionFunc(
		(*PlacementRequestBinderArgs)(nil),
		(*v1alpha1.PlacementRequestBinderArgs)(nil),
		internalToV1alpha1,
	); err != nil {
		return fmt.Errorf("adding internal to v1alpha1 conversion: %v", err)
	}

	return nil
}

// internalToV1alpha1 converts internal PlacementRequestBinderArgs to an
// external one.
func internalToV1alpha1(in, out interface{}, s conversion.Scope) error {
	inArgs := in.(*PlacementRequestBinderArgs)
	outArgs := out.(*v1alpha1.PlacementRequestBinderArgs)
	outArgs.Timeout = inArgs.Timeout
	return nil
}

// v1alpha1ToInternal converts an external PlacementRequestBinderArgs to an
// internal one.
func v1alpha1ToInternal(in, out interface{}, s conversion.Scope) error {
	inArgs := in.(*v1alpha1.PlacementRequestBinderArgs)
	outArgs := out.(*PlacementRequestBinderArgs)
	outArgs.Timeout = inArgs.Timeout
	return nil
}

// init will register both the external and the internal types against the
// needed schemas in the scheduler source code. both versions here declare
// themselves as part of the kubescheduler.config.k8s.io version otherwise
// the scheduler configuration parser won't recognize them. XXX obviously
// this isn't the best way of doing this but it seems like the only way.
func init() {
	for _, scheme := range []*runtime.Scheme{
		scheme.Scheme, v1.GetPluginArgConversionScheme(),
	} {
		scheme.AddKnownTypes(
			SchemeGroupVersion, &PlacementRequestBinderArgs{},
		)
		if err := v1alpha1.AddToScheme(scheme); err != nil {
			panic(err)
		}
		if err := RegisterConversions(scheme); err != nil {
			panic(err)
		}
	}
}
