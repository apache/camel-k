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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/util"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/defaults"
	logger "github.com/apache/camel-k/pkg/util/log"
)

var log = logger.WithName("builder")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Camel K Version: %v", defaults.Version))
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	printVersion()

	c, err := client.NewClient()
	exitOnError(err)

	ctx := cancellable.NewContext()

	build := &v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: os.Args[1],
			Name:      os.Args[2],
		},
	}

	exitOnError(
		c.Get(ctx, types.NamespacedName{Namespace: build.Namespace, Name: build.Name}, build),
	)

	req, err := util.NewRequestForBuild(ctx, c, build)
	exitOnError(err)

	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhaseRunning
	exitOnError(
		util.UpdateBuildStatus(ctx, target, c, log),
	)

	result := builder.New(c).Build(*req)

	exitOnError(
		util.UpdateBuildFromResult(req.C, build, result, c, log),
	)

	switch result.Status {
	case v1alpha1.BuildPhaseSucceeded:
		os.Exit(0)
	default:
		os.Exit(1)
	}
}

func exitOnError(err error) {
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
}
