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

package platformcontroller

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapctrl "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/apache/camel-k/v2/pkg/apis"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/controller"
	"github.com/apache/camel-k/v2/pkg/event"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
)

var log = logutil.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Camel K Platform Controller Version: %v", defaults.Version))
	log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	log.Info(fmt.Sprintf("Camel K Git Commit: %v", defaults.GitCommit))
	log.Info(fmt.Sprintf("Camel K Platform Controller ID: %v", defaults.OperatorID()))

	// Will only appear if DEBUG level has been enabled using the env var LOG_LEVEL
	log.Debug("*** DEBUG level messages will be logged ***")
}

// Run starts the Camel K platform controller.
func Run(healthPort, monitoringPort int32, leaderElection bool, leaderElectionID string) {

	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.

	// The constants specified here are zap specific
	var logLevel zapcore.Level
	logLevelVal, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
		switch strings.ToLower(logLevelVal) {
		case "error":
			logLevel = zapcore.ErrorLevel
		case "info":
			logLevel = zapcore.InfoLevel
		case "debug":
			logLevel = zapcore.DebugLevel
		default:
			customLevel, err := strconv.Atoi(strings.ToLower(logLevelVal))
			exitOnError(err, "Invalid log-level")
			// Need to multiply by -1 to turn logr expected level into zap level
			logLevel = zapcore.Level(int8(customLevel) * -1)
		}
	} else {
		logLevel = zapcore.InfoLevel
	}

	// Use and set atomic level that all following log events are compared with
	// in order to evaluate if a given log level on the event is enabled.
	logf.SetLogger(zapctrl.New(func(o *zapctrl.Options) {
		o.Development = false
		o.Level = zap.NewAtomicLevelAt(logLevel)
	}))

	klog.SetLogger(log.AsLogger())

	_, err := maxprocs.Set(maxprocs.Logger(func(f string, a ...interface{}) { log.Info(fmt.Sprintf(f, a)) }))
	if err != nil {
		log.Error(err, "failed to set GOMAXPROCS from cgroups")
	}

	printVersion()

	watchNamespace, err := getWatchNamespace()
	exitOnError(err, "failed to get watch namespace")

	ctx := signals.SetupSignalHandler()

	cfg, err := config.GetConfig()
	exitOnError(err, "cannot get client config")
	// Increase maximum burst that is used by client-side throttling,
	// to prevent the requests made to apply the bundled Kamelets
	// from being throttled.
	cfg.QPS = 20
	cfg.Burst = 200
	bootstrapClient, err := client.NewClientWithConfig(false, cfg)
	exitOnError(err, "cannot initialize client")

	// We do not rely on the event broadcaster managed by controller runtime,
	// so that we can check the operator has been granted permission to create
	// Events. This is required for the operator to be installable by standard
	// admin users, that are not granted create permission on Events by default.
	broadcaster := record.NewBroadcaster()
	defer broadcaster.Shutdown()

	if ok, err := kubernetes.CheckPermission(ctx, bootstrapClient, corev1.GroupName, "events", watchNamespace, "", "create"); err != nil || !ok {
		// Do not sink Events to the server as they'll be rejected
		broadcaster = event.NewSinkLessBroadcaster(broadcaster)
		exitOnError(err, "cannot check permissions for creating Events")
		log.Info("Event broadcasting is disabled because of missing permissions to create Events")
	}

	platformcontrollerNamespace := platform.GetPlatformControllerNamespace()
	if platformcontrollerNamespace == "" {
		// Fallback to using the watch namespace when the platform controller is not in-cluster.
		// It does not support local (off-cluster) platform controller watching resources globally,
		// in which case it's not possible to determine a namespace.
		platformcontrollerNamespace = watchNamespace
		if platformcontrollerNamespace == "" {
			leaderElection = false
			log.Info("unable to determine namespace for leader election")
		}
	}

	// Set the platform controller container image if it runs in-container
	platform.PlatformControllerImage, err = getPlatformControllerImage(ctx, bootstrapClient)
	exitOnError(err, "cannot get platform controller container image")

	if ok, err := kubernetes.CheckPermission(ctx, bootstrapClient, coordination.GroupName, "leases", platformcontrollerNamespace, "", "create"); err != nil || !ok {
		leaderElection = false
		exitOnError(err, "cannot check permissions for creating Leases")
		log.Info("The platform controller is not granted permissions to create Leases")
	}

	if !leaderElection {
		log.Info("Leader election is disabled!")
	}

	options := cache.Options{
		Namespaces: []string{watchNamespace},
	}

	mgr, err := manager.New(cfg, manager.Options{
		EventBroadcaster:              broadcaster,
		LeaderElection:                leaderElection,
		LeaderElectionNamespace:       platformcontrollerNamespace,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		HealthProbeBindAddress:        ":" + strconv.Itoa(int(healthPort)),
		MetricsBindAddress:            ":" + strconv.Itoa(int(monitoringPort)),
		Cache:                         options,
	})
	exitOnError(err, "")

	log.Info("Configuring manager")
	exitOnError(mgr.AddHealthzCheck("health-probe", healthz.Ping), "Unable add liveness check")
	exitOnError(apis.AddToScheme(mgr.GetScheme()), "")
	ctrlClient, err := client.FromManager(mgr)
	exitOnError(err, "")
	exitOnError(controller.AddToPlatformManager(ctx, mgr, ctrlClient), "")

	log.Info("Installing platform manager resources")
	installCtx, installCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer installCancel()
	install.OperatorStartupOptionalTools(installCtx, bootstrapClient, watchNamespace, platformcontrollerNamespace, log)

	log.Info("Starting the platform manager")
	exitOnError(mgr.Start(ctx), "platform manager exited non-zero")
}

// getWatchNamespace returns the Namespace the platform controller should be watching for changes.
func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(platform.PlatformControllerWatchNamespaceEnvVariable)
	if !found {
		return "", fmt.Errorf("%s must be set", platform.PlatformControllerWatchNamespaceEnvVariable)
	}
	return ns, nil
}

// getPlatformControllerImage returns the image currently used by the running platform controller if present (when running out of cluster, it may be absent).
func getPlatformControllerImage(ctx context.Context, c ctrl.Reader) (string, error) {
	ns := platform.GetPlatformControllerNamespace()
	name := platform.GetOperatorPodName()
	if ns == "" || name == "" {
		return "", nil
	}

	pod := corev1.Pod{}
	if err := c.Get(ctx, ctrl.ObjectKey{Namespace: ns, Name: name}, &pod); err != nil && k8serrors.IsNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	if len(pod.Spec.Containers) == 0 {
		return "", fmt.Errorf("no containers found in platform controller pod")
	}
	return pod.Spec.Containers[0].Image, nil
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}
