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

package cmd

import (
	"github.com/apache/camel-k/v2/pkg/cmd/platformcontroller"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/spf13/cobra"
)

const platformcontrollerCommand = "platformcontroller"

func newCmdPlatformController(rootCmdOptions *RootCmdOptions) (*cobra.Command, *platformcontrollerCmdOptions) {
	options := platformcontrollerCmdOptions{}

	cmd := cobra.Command{
		Use:     "platformcontroller",
		Short:   "Run the Camel K platform controller",
		Long:    `Run the Camel K platform controller`,
		Hidden:  true,
		PreRunE: decode(&options, rootCmdOptions.Flags),
		Run:     options.run,
	}

	cmd.Flags().Int32("health-port", 8081, "The port of the health endpoint")
	cmd.Flags().Int32("monitoring-port", 8080, "The port of the metrics endpoint")
	cmd.Flags().Bool("leader-election", true, "Use leader election")
	cmd.Flags().String("leader-election-id", "", "Use the given ID as the leader election Lease name")

	return &cmd, &options
}

type platformcontrollerCmdOptions struct {
	HealthPort       int32  `mapstructure:"health-port"`
	MonitoringPort   int32  `mapstructure:"monitoring-port"`
	LeaderElection   bool   `mapstructure:"leader-election"`
	LeaderElectionID string `mapstructure:"leader-election-id"`
}

func (o *platformcontrollerCmdOptions) run(_ *cobra.Command, _ []string) {

	leaderElectionID := o.LeaderElectionID
	if leaderElectionID == "" {
		if defaults.OperatorID() != "" {
			leaderElectionID = platform.GetPlatformControllerLockName(defaults.OperatorID())
		} else {
			leaderElectionID = platform.PlatformControllerLockName
		}
	}

	platformcontroller.Run(o.HealthPort, o.MonitoringPort, o.LeaderElection, leaderElectionID)
}
