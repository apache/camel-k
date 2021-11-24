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

package builder

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	logger "github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/patch"
)

const terminationMessagePath = "/dev/termination-log"

var log = logger.WithName("builder")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Camel K Version: %v", defaults.Version))
}

// Run a build resource in the specified namespace.
func Run(namespace string, buildName string, taskName string) {
	logf.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = false
	}))

	rand.Seed(time.Now().UTC().UnixNano())
	printVersion()

	c, err := client.NewClient(false)
	exitOnError(err, "")

	ctx := context.Background()
	cancelOnSignals := contextWithInterrupts(ctx)

	build := &v1.Build{}
	exitOnError(c.Get(cancelOnSignals, types.NamespacedName{Namespace: namespace, Name: buildName}, build), "")

	status := builder.New(c).Build(build).TaskByName(taskName).Do(cancelOnSignals)
	target := build.DeepCopy()
	target.Status = status
	// Let the owning controller decide the resulting phase based on the Pod state.
	// The Pod status acts as the interface with the controller, so that no assumptions
	// is made on the build containers.
	target.Status.Phase = v1.BuildPhaseNone
	// Patch the build status with the result
	p, err := patch.PositiveMergePatch(build, target)
	exitOnError(err, "cannot create merge patch")

	if len(p) > 0 {
		exitOnError(
			c.Status().Patch(ctx, target, ctrl.RawPatch(types.MergePatchType, p)),
			fmt.Sprintf("\n--- patch ---\n%s\n-------------\n", string(p)),
		)
	}

	switch status.Phase {
	case v1.BuildPhaseFailed, v1.BuildPhaseInterrupted, v1.BuildPhaseError:
		log.Error(nil, status.Error)
		// Write the error into the container termination message
		writeTerminationMessage(status.Error)
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func exitOnError(err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}

func writeTerminationMessage(message string) {
	// #nosec G306
	err := ioutil.WriteFile(terminationMessagePath, []byte(message), 0o644)
	if err != nil {
		log.Error(err, "cannot write termination message")
	}
}

// contextWithInterrupts registers for SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func contextWithInterrupts(parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		<-c
		log.Error(nil, "The build has been interrupted")
		// Write the container termination message
		writeTerminationMessage("Pod terminated")
		os.Exit(1)
	}()

	return ctx
}
