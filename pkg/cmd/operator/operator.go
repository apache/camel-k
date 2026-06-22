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
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapctrl "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/apache/camel-k/v2/pkg/apis"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/controller"
	"github.com/apache/camel-k/v2/pkg/controller/synthetic"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
)

var log = logutil.Log.WithName("cmd")

func printVersion() {
	log.Info("Go Version: " + runtime.Version())
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Camel K Operator Version: %v", defaults.Version))
	log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	log.Info(fmt.Sprintf("Camel K Git Commit: %v", defaults.GitCommit))
	log.Info(fmt.Sprintf("Camel K Operator ID: %v", defaults.OperatorID()))

	// Will only appear if DEBUG level has been enabled using the env var LOG_LEVEL
	log.Debug("*** DEBUG level messages will be logged ***")
}

// Run starts the Camel K operator.
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
			int8Lev, err := util.IToInt8(customLevel)
			exitOnError(err, "Invalid log-level")
			// Need to multiply by -1 to turn logr expected level into zap level
			logLevel = zapcore.Level(*int8Lev * -1)
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

	_, err := maxprocs.Set(maxprocs.Logger(func(f string, a ...any) { log.Info(fmt.Sprintf(f, a)) }))
	if err != nil {
		log.Error(err, "failed to set GOMAXPROCS from cgroups")
	}

	printVersion()

	// WATCH_NAMESPACE must be present (it may be empty, which selects global mode).
	if _, err := getWatchNamespace(); err != nil {
		exitOnError(err, "failed to get watch namespace")
	}
	// watchNamespaces is the explicit list configured via WATCH_NAMESPACE (single or comma-separated).
	watchNamespaces := platform.GetWatchNamespaces()

	// watchSelector, when set, enables dynamic discovery of namespaces to watch by label.
	var watchSelector labels.Selector
	if selectorStr := platform.GetWatchNamespaceSelector(); selectorStr != "" {
		parsed, err := labels.Parse(selectorStr)
		exitOnError(err, "invalid "+platform.OperatorWatchNamespaceSelectorEnvVariable)
		watchSelector = parsed
	}

	ctx := signals.SetupSignalHandler()
	// managerCtx lets the operator gracefully restart itself (recomputing its watched namespaces
	// at startup) when the set of dynamically discovered namespaces changes. Cancelling it stops
	// the manager cleanly without terminating the process abnormally; the process then exits 0 and
	// is restarted by Kubernetes.
	managerCtx, restartOperator := context.WithCancel(ctx)
	defer restartOperator()

	cfg, err := config.GetConfig()
	exitOnError(err, "cannot get client config")
	// Increase maximum burst that is used by client-side throttling,
	// to prevent the requests made to apply the bundled Kamelets
	// from being throttled.
	cfg.QPS = 20
	cfg.Burst = 200
	bootstrapClient, err := client.NewClientWithConfig(false, cfg)
	exitOnError(err, "cannot initialize client")

	operatorNamespace := platform.GetOperatorNamespace()
	if operatorNamespace == "" {
		// Fallback to using a watched namespace when the operator is not in-cluster.
		// It does not support local (off-cluster) operator watching resources globally,
		// in which case it's not possible to determine a namespace.
		if len(watchNamespaces) > 0 {
			operatorNamespace = watchNamespaces[0]
		}
		if operatorNamespace == "" {
			leaderElection = false
			log.Info("unable to determine namespace for leader election")
		}
	}

	// Set the operator container image if it runs in-container
	platform.OperatorImage, err = getOperatorImage(ctx, bootstrapClient)
	exitOnError(err, "cannot get operator container image")

	if !leaderElection {
		log.Info("Leader election is disabled!")
	}

	// Determine the set of namespaces to watch. In global mode this stays nil (watch everything);
	// otherwise it is the operator namespace plus any statically configured (WATCH_NAMESPACE)
	// and/or dynamically discovered (WATCH_NAMESPACE_SELECTOR) namespaces the operator can access.
	var watchedNamespaces map[string]bool
	if !platform.IsCurrentOperatorGlobal() {
		watchedNamespaces, err = computeWatchedNamespaces(ctx, bootstrapClient, cfg, bootstrapClient.GetScheme(),
			operatorNamespace, watchNamespaces, watchSelector)
		exitOnError(err, "cannot determine the namespaces to watch")
		log.Info("Operator is watching a defined set of namespaces", "namespaces", sortedKeys(watchedNamespaces))
	}

	hasIntegrationLabel, err := labels.NewRequirement(v1.IntegrationLabel, selection.Exists, []string{})
	exitOnError(err, "cannot create Integration label selector")
	labelsSelector := labels.NewSelector().Add(*hasIntegrationLabel)

	selector := cache.ByObject{
		Label: labelsSelector,
	}

	if !platform.IsCurrentOperatorGlobal() {
		selector.Namespaces = toCacheNamespaces(watchedNamespaces)
	}

	selectors := map[ctrl.Object]cache.ByObject{
		&corev1.Pod{}:        selector,
		&appsv1.Deployment{}: selector,
		&batchv1.Job{}:       selector,
	}

	if ok, err := kubernetes.IsAPIResourceInstalled(bootstrapClient, servingv1.SchemeGroupVersion.String(), reflect.TypeFor[servingv1.Service]().Name()); ok && err == nil {
		selectors[&servingv1.Service{}] = selector
	}

	if ok, err := kubernetes.IsAPIResourceInstalled(bootstrapClient, batchv1.SchemeGroupVersion.String(), reflect.TypeFor[batchv1.CronJob]().Name()); ok && err == nil {
		selectors[&batchv1.CronJob{}] = selector
	}

	options := cache.Options{
		ByObject: selectors,
	}

	if !platform.IsCurrentOperatorGlobal() {
		options.DefaultNamespaces = toCacheNamespaces(watchedNamespaces)
	}

	mgr, err := manager.New(cfg, manager.Options{
		LeaderElection:                leaderElection,
		LeaderElectionNamespace:       operatorNamespace,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		HealthProbeBindAddress:        ":" + strconv.Itoa(int(healthPort)),
		Metrics:                       metricsserver.Options{BindAddress: ":" + strconv.Itoa(int(monitoringPort))},
		Cache:                         options,
	})
	exitOnError(err, "")

	log.Info("Configuring manager")
	exitOnError(mgr.AddHealthzCheck("health-probe", healthz.Ping), "Unable add liveness check")
	exitOnError(apis.AddToScheme(mgr.GetScheme()), "")
	ctrlClient, err := client.FromManager(mgr)
	exitOnError(err, "")
	exitOnError(controller.AddToManager(ctx, mgr, ctrlClient), "")

	// When dynamic namespace discovery is enabled, watch Namespace objects matching the selector
	// and gracefully restart the operator whenever the watchable namespace set changes.
	if watchSelector != nil {
		exitOnError(mgr.Add(&namespaceWatcher{
			config:            cfg,
			scheme:            mgr.GetScheme(),
			reviewer:          bootstrapClient,
			selector:          watchSelector,
			operatorNamespace: operatorNamespace,
			staticNamespaces:  watchNamespaces,
			current:           watchedNamespaces,
			requestRestart:    restartOperator,
		}), "cannot add the dynamic namespace watcher")
		log.Info("Dynamic namespace discovery is enabled", "selector", watchSelector.String())
	}

	log.Info("Installing operator resources")
	installCtx, installCancel := context.WithTimeout(ctx, 1*time.Minute)
	defer installCancel()
	install.OperatorStartupOptionalTools(installCtx, bootstrapClient, platform.GetOperatorWatchNamespace(), operatorNamespace, log)

	synthEnvVal, synth := os.LookupEnv("CAMEL_K_SYNTHETIC_INTEGRATIONS")
	if synth && synthEnvVal == "true" {
		log.Info("Starting the synthetic Integration manager. " +
			"WARNING: this is a deprecated feature and will be removed in future versions, use Camel Dashboard project instead.")
		exitOnError(synthetic.ManageSyntheticIntegrations(ctx, ctrlClient, mgr.GetCache()), "synthetic Integration manager error")
	}
	log.Info("Starting the manager")
	// managerCtx is cancelled either by a process signal (graceful shutdown) or by the namespace
	// watcher requesting a restart. In both cases Start returns nil and the process exits 0; on a
	// restart request Kubernetes brings the operator back up and it recomputes its watched set.
	exitOnError(mgr.Start(managerCtx), "manager exited non-zero")
}

// toCacheNamespaces converts a set of namespace names into the per-namespace cache configuration
// expected by controller-runtime. An empty/nil input yields an empty map.
func toCacheNamespaces(namespaces map[string]bool) map[string]cache.Config {
	namespacesSelector := make(map[string]cache.Config, len(namespaces))
	for ns := range namespaces {
		namespacesSelector[ns] = cache.Config{}
	}

	return namespacesSelector
}

// getWatchNamespace returns the Namespace the operator should be watching for changes.
func getWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(platform.OperatorWatchNamespaceEnvVariable)
	if !found {
		return "", fmt.Errorf("%s must be set", platform.OperatorWatchNamespaceEnvVariable)
	}

	return ns, nil
}

// getOperatorImage returns the image currently used by the running operator if present (when running out of cluster, it may be absent).
func getOperatorImage(ctx context.Context, c ctrl.Reader) (string, error) {
	ns := platform.GetOperatorNamespace()
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
		return "", errors.New("no containers found in operator pod")
	}

	return pod.Spec.Containers[0].Image, nil
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}
