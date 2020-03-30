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
	"time"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/install"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/controller"
	"github.com/apache/camel-k/pkg/util/defaults"
)

var log = logf.Log.WithName("cmd")

// GitCommit --
var GitCommit string

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Operator SDK Version: %v", sdkVersion.Version))
	log.Info(fmt.Sprintf("Buildah Version: %v", defaults.BuildahVersion))
	log.Info(fmt.Sprintf("Kaniko Version: %v", defaults.KanikoVersion))
	log.Info(fmt.Sprintf("Camel K Operator Version: %v", defaults.Version))
	log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	log.Info(fmt.Sprintf("Camel K Git Commit: %v", GitCommit))
}

// Run starts the Camel K operator
func Run() {
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

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	err = leader.Become(context.TODO(), "camel-k-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	defer r.Unset() // nolint: errcheck

	// Configure an event broadcaster
	c, err := client.NewClient(false)
	if err != nil {
		log.Error(err, "cannot initialize client")
		os.Exit(1)
	}

	// Configure event broadcaster
	var eventBroadcaster record.EventBroadcaster
	// nolint: gocritic
	if ok, err := kubernetes.CheckPermission(c, corev1.GroupName, "events", namespace, "", "create"); err != nil {
		log.Error(err, "cannot check permissions for configuring event broadcaster")
	} else if !ok {
		log.Info("Event broadcasting to Kubernetes is disabled because of missing permissions to create events")
	} else {
		eventBroadcaster = record.NewBroadcaster()
		//eventBroadcaster.StartLogging(camellog.WithName("events").Infof)
		eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: c.CoreV1().Events(namespace)})
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:        namespace,
		EventBroadcaster: eventBroadcaster,
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Try to register the OpenShift CLI Download link if possible
	installCtx, _ := context.WithTimeout(context.TODO(), 30 * time.Second)
	if err := install.OpenShiftConsoleDownloadLink(installCtx, c); err != nil {
		log.Info("Cannot install OpenShift CLI download link: skipping.")
		log.V(8).Info("Error while installing OpenShift CLI download link", "error", err)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

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
