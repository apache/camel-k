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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	coordination "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/apache/camel-k/pkg/apis"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/controller"
	"github.com/apache/camel-k/pkg/event"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Buildah Version: %v", defaults.BuildahVersion))
	log.Info(fmt.Sprintf("Kaniko Version: %v", defaults.KanikoVersion))
	log.Info(fmt.Sprintf("Camel K Operator Version: %v", defaults.Version))
	log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	log.Info(fmt.Sprintf("Camel K Git Commit: %v", defaults.GitCommit))
}

// Run starts the Camel K operator.
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

	// Set the operator container image if it runs in-container
	platform.OperatorImage, err = getOperatorImage(context.TODO(), c)
	exitOnError(err, "cannot get operator container image")

	if ok, err := kubernetes.CheckPermission(context.TODO(), c, coordination.GroupName, "leases", operatorNamespace, "", "create"); err != nil || !ok {
		leaderElection = false
		exitOnError(err, "cannot check permissions for creating Leases")
		log.Info("The operator is not granted permissions to create Leases")
	}

	if !leaderElection {
		log.Info("Leader election is disabled!")
	}

	hasIntegrationLabel, err := labels.NewRequirement(v1.IntegrationLabel, selection.Exists, []string{})
	exitOnError(err, "cannot create Integration label selector")
	selector := labels.NewSelector().Add(*hasIntegrationLabel)

	mgr, err := manager.New(c.GetConfig(), manager.Options{
		Namespace:                     watchNamespace,
		EventBroadcaster:              broadcaster,
		LeaderElection:                leaderElection,
		LeaderElectionNamespace:       operatorNamespace,
		LeaderElectionID:              platform.OperatorLockName,
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		HealthProbeBindAddress:        ":" + strconv.Itoa(int(healthPort)),
		MetricsBindAddress:            ":" + strconv.Itoa(int(monitoringPort)),
		NewCache: cache.BuilderWithOptions(
			cache.Options{
				SelectorsByObject: cache.SelectorsByObject{
					&corev1.Pod{}:           {Label: selector},
					&appsv1.Deployment{}:    {Label: selector},
					&batchv1beta1.CronJob{}: {Label: selector},
					&batchv1.Job{}:          {Label: selector},
					&servingv1.Service{}:    {Label: selector},
				},
			},
		),
	})
	exitOnError(err, "")

	exitOnError(
		mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "status.phase",
			func(obj ctrl.Object) []string {
				return []string{string(obj.(*corev1.Pod).Status.Phase)}
			}),
		"unable to set up field indexer for status.phase: %v",
	)

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
		return "", fmt.Errorf("no containers found in operator pod")
	}
	return pod.Spec.Containers[0].Image, nil
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}
