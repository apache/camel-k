/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manager

import (
	"context"
	"strconv"
	"time"

	"github.com/apache/camel-k/v2/pkg/apis"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/event"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type Manager interface {
	PrintVersion()
	CreateBootstrapClient(cfg *rest.Config) (client.Client, string, error)
	CreateEventBroadcaster(ctx context.Context, bootstrapClient client.Client) (record.EventBroadcaster, string, error)
	GetControllerNamespaceAndLeaderElection(ctx context.Context, bootstrapClient client.Client, leaderElection bool) (string, bool, string, error)
	GetManagerOptions(bootstrapClient client.Client) (cache.Options, string, error)
	CreateManager(ctx context.Context, healthPort int32, monitoringPort int32, leaderElection bool, leaderElectionID string, cfg *rest.Config, eventBroadcaster record.EventBroadcaster, controllerNamespace string, options cache.Options) (manager.Manager, client.Client, string, error)
	ControllerPreStartResourcesInit(ctx context.Context, initCtx context.Context, bootstrapClient client.Client, controllerNamespace string, ctrlClient client.Client, mgr manager.Manager) (string, error)
}

func NewControllerCmd(controllerManager Manager, log logutil.Logger) *ControllerCmd {
	return &ControllerCmd{
		controllerManager: controllerManager,
		log:               log,
	}
}

type ControllerCmd struct {
	controllerManager Manager
	log               logutil.Logger
}

func (c ControllerCmd) Run(healthPort, monitoringPort int32, leaderElection bool, leaderElectionID string) (string, error) {
	errMessage, err := setMaxprocs(c.log)
	if err != nil {
		return errMessage, err
	}

	c.controllerManager.PrintVersion()
	// Will only appear if DEBUG level has been enabled using the env var LOG_LEVEL
	c.log.Debug("*** DEBUG level messages will be logged ***")

	cfg, err := config.GetConfig()
	if err != nil {
		return "cannot get client config", err
	}
	bootstrapClient, errMessage, err := c.controllerManager.CreateBootstrapClient(cfg)
	if err != nil {
		return errMessage, err
	}

	ctx := signals.SetupSignalHandler()

	eventBroadcaster, errMessage, err := c.controllerManager.CreateEventBroadcaster(ctx, bootstrapClient)
	if eventBroadcaster != nil {
		defer eventBroadcaster.Shutdown()
	}
	if err != nil {
		return errMessage, err
	}

	controllerNamespace, leaderElection, errMessage, err := c.controllerManager.GetControllerNamespaceAndLeaderElection(ctx, bootstrapClient, leaderElection)
	if err != nil {
		return errMessage, err
	}
	if !leaderElection {
		c.log.Info("Leader election is disabled!")
	}

	errMessage, err = setOperatorImage(ctx, bootstrapClient, controllerNamespace)
	if err != nil {
		return errMessage, err
	}

	options, errMessage, err := c.controllerManager.GetManagerOptions(bootstrapClient)
	if err != nil {
		return errMessage, err
	}

	mgr, ctrlClient, errMessage, err := c.controllerManager.CreateManager(ctx, healthPort, monitoringPort, leaderElection, leaderElectionID, cfg, eventBroadcaster, controllerNamespace, options)
	if err != nil {
		return errMessage, err
	}

	initCtx, initCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer initCancel()

	errMessage, err = c.controllerManager.ControllerPreStartResourcesInit(ctx, initCtx, bootstrapClient, controllerNamespace, ctrlClient, mgr)
	if err != nil {
		return errMessage, err
	}

	c.log.Info("Starting the manager")
	return "manager exited non-zero", mgr.Start(ctx)
}

type BaseManager struct {
	Log                 logutil.Logger
	WatchNamespace      string
	ControllerNamespace string
	AddToManager        func(ctx context.Context, manager manager.Manager, client client.Client) error
}

func (bm BaseManager) CreateBootstrapClient(cfg *rest.Config) (client.Client, string, error) {
	// Increase maximum burst that is used by client-side throttling,
	// to prevent the requests made to apply the bundled Kamelets
	// from being throttled.
	cfg.QPS = 20
	cfg.Burst = 200
	bootstrapClient, err := client.NewClientWithConfig(false, cfg)
	if err != nil {
		return nil, "cannot initialize client", err
	}

	return bootstrapClient, "", nil
}

func (bm BaseManager) CreateEventBroadcaster(ctx context.Context, bootstrapClient client.Client) (record.EventBroadcaster, string, error) {
	// We do not rely on the event broadcaster managed by controller runtime,
	// so that we can check the operator has been granted permission to create
	// Events. This is required for the operator to be installable by standard
	// admin users, that are not granted create permission on Events by default.
	broadcaster := record.NewBroadcaster()

	if ok, err := kubernetes.CheckPermission(ctx, bootstrapClient, corev1.GroupName, "events", bm.WatchNamespace, "", "create"); err != nil || !ok {
		// Do not sink Events to the server as they'll be rejected
		broadcaster = event.NewSinkLessBroadcaster(broadcaster)
		if err != nil {
			bm.Log.Info("Event broadcasting is disabled because of missing permissions to create Events")
		}
		return broadcaster, "cannot check permissions for creating Events", err
	}

	return broadcaster, "", nil
}

func (bm BaseManager) GetControllerNamespaceAndLeaderElection(ctx context.Context, bootstrapClient client.Client, leaderElection bool) (string, bool, string, error) {
	controllerNamespace := bm.ControllerNamespace
	if controllerNamespace == "" {
		// Fallback to using the watch namespace when the operator is not in-cluster.
		// It does not support local (off-cluster) operator watching resources globally,
		// in which case it's not possible to determine a namespace.
		controllerNamespace = bm.WatchNamespace
		if controllerNamespace == "" {
			leaderElection = false
			bm.Log.Info("unable to determine namespace for leader election")
		}
	}

	if ok, err := kubernetes.CheckPermission(ctx, bootstrapClient, coordination.GroupName, "leases", controllerNamespace, "", "create"); err != nil || !ok {
		leaderElection = false
		if err != nil {
			return controllerNamespace, leaderElection, "cannot check permissions for creating Leases", err
		}
		bm.Log.Info("The operator is not granted permissions to create Leases")
	}

	return controllerNamespace, leaderElection, "", nil
}

func (bm BaseManager) CreateManager(ctx context.Context, healthPort int32, monitoringPort int32, leaderElection bool, leaderElectionID string, cfg *rest.Config, eventBroadcaster record.EventBroadcaster, controllerNamespace string, options cache.Options) (manager.Manager, client.Client, string, error) {
	mgr, err := manager.New(cfg, manager.Options{
		// it is not clear in favor of what this is deprecated, opened an issue https://github.com/kubernetes-sigs/controller-runtime/issues/2694
		EventBroadcaster:              eventBroadcaster,
		LeaderElection:                leaderElection,
		LeaderElectionNamespace:       controllerNamespace,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		HealthProbeBindAddress:        ":" + strconv.Itoa(int(healthPort)),
		MetricsBindAddress:            ":" + strconv.Itoa(int(monitoringPort)),
		Cache:                         options,
	})
	if err != nil {
		return nil, nil, "", err
	}

	bm.Log.Info("Configuring manager")
	if err := mgr.AddHealthzCheck("health-probe", healthz.Ping); err != nil {
		return nil, nil, "Unable add liveness check", err
	}
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, nil, "", err
	}
	ctrlClient, err := client.FromManager(mgr)
	if err != nil {
		return nil, nil, "", err
	}
	if err := bm.AddToManager(ctx, mgr, ctrlClient); err != nil {
		return nil, nil, "", err
	}

	return mgr, ctrlClient, "", nil
}
