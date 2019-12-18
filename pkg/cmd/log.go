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
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newCmdLog(rootCmdOptions *RootCmdOptions) (*cobra.Command, *logCmdOptions) {
	options := logCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "log integration",
		Short:   "Print the logs of an integration",
		Long:    `Print the logs of an integration.`,
		Args:    options.validate,
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd, &options
}

type logCmdOptions struct {
	*RootCmdOptions
}

func (o *logCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("accepts 1 arg, received %d", len(args))
	}

	return nil
}

func (o *logCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	integration := v1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.IntegrationKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      args[0],
		},
	}
	key := k8sclient.ObjectKey{
		Namespace: o.Namespace,
		Name:      args[0],
	}

	if err := c.Get(o.Context, key, &integration); err != nil {
		return err
	}
	if err := k8slog.Print(o.Context, c, &integration, cmd.OutOrStdout()); err != nil {
		return err
	}

	// Let's add a Wait point, otherwise the script terminates
	<-o.Context.Done()

	return nil
}
