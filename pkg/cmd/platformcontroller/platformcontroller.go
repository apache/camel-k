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

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/cmd/manager"
	"github.com/apache/camel-k/v2/pkg/controller"
	"github.com/apache/camel-k/v2/pkg/install"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	logutil "github.com/apache/camel-k/v2/pkg/util/log"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlManager "sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logutil.Log.WithName("platformcontroller")

type platformControllerManager struct {
	manager.BaseManager
}

// Run starts the Camel K platform controller.
func Run(healthPort, monitoringPort int32, leaderElection bool, leaderElectionID string) {
	flag.Parse()

	errMessage, err := logutil.LoggerSetup(&log)
	if err != nil {
		log.Error(err, errMessage)
		os.Exit(1)
	}
	watchNamespace, err := manager.GetWatchNamespace(platform.PlatformControllerWatchNamespaceEnvVariable)
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	pcm := platformControllerManager{
		manager.BaseManager{
			Log:                 log,
			WatchNamespace:      watchNamespace,
			ControllerNamespace: platform.GetPlatformControllerNamespace(),
			AddToManager:        controller.AddToPlatformManager,
		},
	}

	controllerCmd := manager.NewControllerCmd(pcm, log)

	errMessage, err = controllerCmd.Run(healthPort, monitoringPort, leaderElection, leaderElectionID)
	if err != nil {
		log.Error(err, errMessage)
		os.Exit(1)
	}
}

func (pcm platformControllerManager) PrintVersion() {
	pcm.Log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	pcm.Log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	pcm.Log.Info(fmt.Sprintf("Camel K Platform Controller Version: %v", defaults.Version))
	pcm.Log.Info(fmt.Sprintf("Camel K Git Commit: %v", defaults.GitCommit))
	pcm.Log.Info(fmt.Sprintf("Camel K Operator ID: %v", defaults.OperatorID()))
}

func (pcm platformControllerManager) GetManagerOptions(bootstrapClient client.Client) (cache.Options, string, error) {
	options := cache.Options{}

	if !platform.IsCurrentOperatorGlobal() {
		options = cache.Options{
			DefaultNamespaces: map[string]cache.Config{pcm.WatchNamespace: {}, pcm.ControllerNamespace: {}},
		}
	}

	return options, "", nil
}

func (pcm platformControllerManager) ControllerPreStartResourcesInit(ctx context.Context, initCtx context.Context, bootstrapClient client.Client, controllerNamespace string, ctrlClient client.Client, mgr ctrlManager.Manager) (string, error) {
	pcm.Log.Info("Installing platform controller resources")
	install.OperatorStartupOptionalTools(initCtx, bootstrapClient, pcm.WatchNamespace, controllerNamespace, log)

	return "", nil
}
