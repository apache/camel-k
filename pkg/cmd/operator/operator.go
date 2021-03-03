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
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/operator-framework/operator-lib/leader"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/controller"
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
func Run(healthPort, monitoringPort int32) {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = false
	}))

	printVersion()

	namespace, err := getWatchNamespace()
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the API server
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), platform.OperatorLockName)
	if err != nil {
		if err == leader.ErrNoNamespace {
			log.Info("Local run detected, leader election is disabled")
		} else {
			log.Error(err, "")
			os.Exit(1)
		}
	}

	// Configure an event broadcaster
	c, err := client.NewClient(false)
	if err != nil {
		log.Error(err, "cannot initialize client")
		os.Exit(1)
	}

	// Configure event broadcaster
	var eventBroadcaster record.EventBroadcaster
	// nolint: gocritic
	if ok, err := kubernetes.CheckPermission(context.TODO(), c, corev1.GroupName, "events", namespace, "", "create"); err != nil {
		log.Error(err, "cannot check permissions for configuring event broadcaster")
	} else if !ok {
		log.Info("Event broadcasting to Kubernetes is disabled because of missing permissions to create events")
	} else {
		eventBroadcaster = record.NewBroadcaster()
		eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: c.CoreV1().Events(namespace)})
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:              namespace,
		EventBroadcaster:       eventBroadcaster,
		HealthProbeBindAddress: ":" + strconv.Itoa(int(healthPort)),
		MetricsBindAddress:     ":" + strconv.Itoa(int(monitoringPort)),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add health check
	if err := mgr.AddHealthzCheck("health-probe", healthz.Ping); err != nil {
		log.Error(err, "Unable add liveness check")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Try to register the OpenShift CLI Download link if possible
	installCtx, installCancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer installCancel()
	operatorNamespace := os.Getenv("NAMESPACE")
	install.OperatorStartupOptionalTools(installCtx, c, namespace, operatorNamespace, log)

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited non-zero")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}
