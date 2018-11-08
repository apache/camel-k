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
	"context"
	"math/rand"
	"runtime"
	"time"

	"github.com/apache/camel-k/pkg/stub"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	_ "github.com/apache/camel-k/pkg/util/knative"
	_ "github.com/apache/camel-k/pkg/util/openshift"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const resyncPeriod = time.Duration(5) * time.Second

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func watch(resource string, kind string, namespace string, resyncPeriod time.Duration) {
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	printVersion()

	sdk.ExposeMetricsPort()

	resource := "camel.apache.org/v1alpha1"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}

	ctx := context.TODO()

	watch(resource, "Integration", namespace, resyncPeriod)
	watch(resource, "IntegrationContext", namespace, resyncPeriod)
	watch(resource, "IntegrationPlatform", namespace, resyncPeriod)

	sdk.Handle(stub.NewHandler(ctx, namespace))
	sdk.Run(ctx)
}
