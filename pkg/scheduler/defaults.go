package scheduler

const (
	// SchedulerName is the name of the scheduler as deployed in the
	// cluster. This scheduler operates only on pods who specify the
	// right schedulerName property.
	SchedulerName = "placement-request-scheduler"

	// PluginName is the name of the plugin as used in the scheduler
	// configuration file.
	PluginName = "PlacementRequestBinder"
)
