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
	"github.com/spf13/cobra"

	"github.com/apache/camel-k/pkg/cmd/operator"
)

const operatorCommand = "operator"

func newCmdOperator() (*cobra.Command, *operatorCmdOptions) {
	options := operatorCmdOptions{}

	cmd := cobra.Command{
		Use:     "operator",
		Short:   "Run the Camel K operator",
		Long:    `Run the Camel K operator`,
		Hidden:  true,
		PreRunE: decode(&options),
		Run:     options.run,
	}

	cmd.Flags().Int32("health-port", 8081, "The port of the health endpoint")
	cmd.Flags().Int32("monitoring-port", 8080, "The port of the metrics endpoint")
	cmd.Flags().Bool("leader-election", true, "Use leader election")

	return &cmd, &options
}

type operatorCmdOptions struct {
	HealthPort     int32 `mapstructure:"health-port"`
	MonitoringPort int32 `mapstructure:"monitoring-port"`
	LeaderElection bool  `mapstructure:"leader-election"`
}

func (o *operatorCmdOptions) run(_ *cobra.Command, _ []string) {
	operator.Run(o.HealthPort, o.MonitoringPort, o.LeaderElection)
}
