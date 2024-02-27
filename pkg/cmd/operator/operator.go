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
	"os"
	"reflect"
	"runtime"

	"github.com/apache/camel-k/v2/pkg/cmd/manager"
	"github.com/apache/camel-k/v2/pkg/controller"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlManager "sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/controller/synthetic"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
)

var log = logutil.Log.WithName("operator")

type operatorManager struct {
	manager.BaseManager
}

// Run starts the Camel K operator.
func Run(healthPort, monitoringPort int32, leaderElection bool, leaderElectionID string) {
	flag.Parse()

	errMessage, err := logutil.LoggerSetup(&log)
	if err != nil {
		log.Error(err, errMessage)
		os.Exit(1)
	}
	watchNamespace, err := manager.GetWatchNamespace(platform.OperatorWatchNamespaceEnvVariable)
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	om := operatorManager{
		manager.BaseManager{
			Log:                 log,
			WatchNamespace:      watchNamespace,
			ControllerNamespace: platform.GetOperatorNamespace(),
			AddToManager:        controller.AddToManager,
		},
	}

	controllerCmd := manager.NewControllerCmd(om, log)

	errMessage, err = controllerCmd.Run(healthPort, monitoringPort, leaderElection, leaderElectionID)
	if err != nil {
		log.Error(err, errMessage)
		os.Exit(1)
	}
}

func (om operatorManager) PrintVersion() {
	om.Log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	om.Log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	om.Log.Info(fmt.Sprintf("Camel K Operator Version: %v", defaults.Version))
	om.Log.Info(fmt.Sprintf("Camel K Default Runtime Version: %v", defaults.DefaultRuntimeVersion))
	om.Log.Info(fmt.Sprintf("Camel K Git Commit: %v", defaults.GitCommit))
	om.Log.Info(fmt.Sprintf("Camel K Operator ID: %v", defaults.OperatorID()))
}

func (om operatorManager) GetManagerOptions(bootstrapClient client.Client) (cache.Options, string, error) {
	hasIntegrationLabel, err := labels.NewRequirement(v1.IntegrationLabel, selection.Exists, []string{})
	if err != nil {
		return cache.Options{}, "cannot create Integration label selector", err
	}
	labelsSelector := labels.NewSelector().Add(*hasIntegrationLabel)

	selector := cache.ByObject{
		Label: labelsSelector,
	}

	if !platform.IsCurrentOperatorGlobal() {
		selector = cache.ByObject{
			Label:      labelsSelector,
			Namespaces: getNamespacesSelector(om.ControllerNamespace, om.WatchNamespace),
		}
	}

	selectors := map[ctrl.Object]cache.ByObject{
		&corev1.Pod{}:        selector,
		&appsv1.Deployment{}: selector,
		&batchv1.Job{}:       selector,
	}

	if ok, err := kubernetes.IsAPIResourceInstalled(bootstrapClient, servingv1.SchemeGroupVersion.String(), reflect.TypeOf(servingv1.Service{}).Name()); ok && err == nil {
		selectors[&servingv1.Service{}] = selector
	}

	if ok, err := kubernetes.IsAPIResourceInstalled(bootstrapClient, batchv1.SchemeGroupVersion.String(), reflect.TypeOf(batchv1.CronJob{}).Name()); ok && err == nil {
		selectors[&batchv1.CronJob{}] = selector
	}

	options := cache.Options{
		ByObject: selectors,
	}

	if !platform.IsCurrentOperatorGlobal() {
		options.DefaultNamespaces = getNamespacesSelector(om.ControllerNamespace, om.WatchNamespace)
	}

	return options, "", nil
}

func (om operatorManager) ControllerPreStartResourcesInit(ctx context.Context, initCtx context.Context, bootstrapClient client.Client, controllerNamespace string, ctrlClient client.Client, mgr ctrlManager.Manager) (string, error) {
	om.Log.Info("Installing operator resources")
	install.TryRegisterOpenShiftConsoleDownloadLink(initCtx, bootstrapClient, log)
	if err := findOrCreateIntegrationPlatform(initCtx, bootstrapClient, controllerNamespace); err != nil {
		return "failed to create integration platform", err
	}

	synthEnvVal, synth := os.LookupEnv("CAMEL_K_SYNTHETIC_INTEGRATIONS")
	if synth && synthEnvVal == "true" {
		om.Log.Info("Starting the synthetic Integration manager")
		if err := synthetic.ManageSyntheticIntegrations(ctx, ctrlClient, mgr.GetCache()); err != nil {
			return "synthetic Integration manager error", err
		}
	} else {
		om.Log.Info("Synthetic Integration manager not configured, skipping")
	}

	return "", nil
}

func getNamespacesSelector(operatorNamespace string, watchNamespace string) map[string]cache.Config {
	namespacesSelector := map[string]cache.Config{
		operatorNamespace: {},
	}
	if operatorNamespace != watchNamespace {
		namespacesSelector[watchNamespace] = cache.Config{}
	}
	return namespacesSelector
}

// findOrCreateIntegrationPlatform create default integration platform in operator namespace if not already exists.
func findOrCreateIntegrationPlatform(ctx context.Context, c client.Client, operatorNamespace string) error {
	operatorID := defaults.OperatorID()
	var platformName string
	if operatorID != "" {
		platformName = operatorID
	} else {
		platformName = platform.DefaultPlatformName
	}

	if pl, err := kubernetes.GetIntegrationPlatform(ctx, c, platformName, operatorNamespace); pl == nil || k8serrors.IsNotFound(err) {
		defaultPlatform := v1.NewIntegrationPlatform(operatorNamespace, platformName)

		if defaultPlatform.Labels == nil {
			defaultPlatform.Labels = make(map[string]string)
		}
		defaultPlatform.Labels["app"] = "camel-k"
		defaultPlatform.Labels["camel.apache.org/platform.generated"] = "true"

		if operatorID != "" {
			defaultPlatform.SetOperatorID(operatorID)
		}

		if _, err := c.CamelV1().IntegrationPlatforms(operatorNamespace).Create(ctx, &defaultPlatform, metav1.CreateOptions{}); err != nil {
			return err
		}

		// Make sure that IntegrationPlatform installed in operator namespace can be seen by others
		if err := install.IntegrationPlatformViewerRole(ctx, c, operatorNamespace); err != nil && !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("error while installing global IntegrationPlatform viewer role: %w", err)
		}
	} else {
		return err
	}

	return nil
}
