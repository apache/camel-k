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

package operator

import (
	"context"
	"flag"
	"fmt"
	"k8s.io/klog/v2"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/controller"
	"github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

var log = logf.Log.WithName("cmd")

// GitCommit --
var GitCommit string

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Buildah Version: %v", defaults.BuildahVersion))
	log.Info(fmt.Sprintf("Kaniko Version: %v", defaults.KanikoVersion))
	log.Info(fmt.Sprintf("Camel K Operator Version: %v", defaults.Version))
	log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	log.Info(fmt.Sprintf("Camel K Git Commit: %v", GitCommit))
}

// Run starts the Camel K operator
func Run(healthPort, monitoringPort int32, leaderElection bool) {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = false
	}))

	klog.SetLogger(log)

	printVersion()

	watchNamespace, err := getWatchNamespace()
	exitOnError(err, "failed to get watch namespace")

	c, err := client.NewClient(false)
	exitOnError(err, "cannot initialize client")

	// We do not rely on the event broadcaster managed by controller runtime,
	// so that we can check the operator has been granted permission to create
	// Events. This is required for the operator to be installable by standard
	// admin users, that are not granted create permission on Events by default.
	broadcaster := record.NewBroadcaster()
	defer broadcaster.Shutdown()
	// nolint: gocritic
	if ok, err := kubernetes.CheckPermission(context.TODO(), c, corev1.GroupName, "events", watchNamespace, "", "create"); err != nil || !ok {
		// Do not sink Events to the server as they'll be rejected
		broadcaster = event.NewSinkLessBroadcaster(broadcaster)
		exitOnError(err, "cannot check permissions for creating Events")
		log.Info("Event broadcasting is disabled because of missing permissions to create Events")
	}

	operatorNamespace := platform.GetOperatorNamespace()
	if operatorNamespace == "" {
		// Fallback to using the watch namespace when the operator is not in-cluster.
		// It does not support local (off-cluster) operator watching resources globally,
		// in which case it's not possible to determine a namespace.
		operatorNamespace = watchNamespace
		if operatorNamespace == "" {
			leaderElection = false
			log.Info("unable to determine namespace for leader election")
		}
	}

	if ok, err := kubernetes.CheckPermission(context.TODO(), c, coordination.GroupName, "leases", operatorNamespace, "", "create"); err != nil || !ok {
		leaderElection = false
		exitOnError(err, "cannot check permissions for creating Leases")
		log.Info("The operator is not granted permissions to create Leases")
	}

	if !leaderElection {
		log.Info("Leader election is disabled!")
	}

	mgr, err := ctrl.NewManager(c.GetConfig(), ctrl.Options{
		Namespace:                     watchNamespace,
		EventBroadcaster:              broadcaster,
		LeaderElection:                leaderElection,
		LeaderElectionNamespace:       operatorNamespace,
		LeaderElectionID:              platform.OperatorLockName,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		HealthProbeBindAddress:        ":" + strconv.Itoa(int(healthPort)),
		MetricsBindAddress:            ":" + strconv.Itoa(int(monitoringPort)),
	})
	exitOnError(err, "")

	log.Info("Configuring manager")
	exitOnError(mgr.AddHealthzCheck("health-probe", healthz.Ping), "Unable add liveness check")
	exitOnError(apis.AddToScheme(mgr.GetScheme()), "")
	exitOnError(controller.AddToManager(mgr), "")

	log.Info("Installing operator resources")
	installCtx, installCancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer installCancel()
	install.OperatorStartupOptionalTools(installCtx, c, watchNamespace, operatorNamespace, log)

	log.Info("Starting the manager")
	exitOnError(mgr.Start(signals.SetupSignalHandler()), "manager exited non-zero")
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(platform.OperatorWatchNamespaceEnvVariable)
	if !found {
		return "", fmt.Errorf("%s must be set", platform.OperatorWatchNamespaceEnvVariable)
	}
	return ns, nil
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}
