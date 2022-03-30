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
	"errors"
	"fmt"
	"time"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
		Aliases: []string{"logs"},
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
		return errors.New("log expects an integration name argument")
	}

	return nil
}

func (o *logCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	integrationID := args[0]

	integration := v1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.IntegrationKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			Name:      integrationID,
		},
	}
	key := k8sclient.ObjectKey{
		Namespace: o.Namespace,
		Name:      integrationID,
	}

	pollTimeout := 600 * time.Second // 10 minutes should be adequate for a timeout
	pollInterval := 2 * time.Second
	currLogMsg := ""
	newLogMsg := ""

	err = wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		//
		// Reduce repetition of messages by tracking the last message
		// and checking if its different from the new message
		//
		if newLogMsg != currLogMsg {
			fmt.Fprintln(cmd.OutOrStdout(), newLogMsg)
			currLogMsg = newLogMsg
		}

		//
		// Try and find the integration
		//
		err = c.Get(o.Context, key, &integration)
		if err != nil && !k8errors.IsNotFound(err) {
			// different error so return
			return false, err
		}

		if k8errors.IsNotFound(err) {
			//
			// Don't have an integration yet so log and wait
			//
			newLogMsg = fmt.Sprintf("Integration '%s' not yet available. Will keep checking ...", integrationID)
			return false, nil
		}

		//
		// Found the integration so check its status using its phase
		//
		phase := integration.Status.Phase
		switch phase {
		case "Running":
			//
			// Found the running integration so step over to scraping its pod log
			//
			fmt.Fprintf(cmd.OutOrStdout(), "Integration '%s' is now running. Showing log ...\n", integrationID)
			if err := k8slog.Print(o.Context, cmd, c, &integration, cmd.OutOrStdout()); err != nil {
				return false, err
			}

			return true, nil
		case "Building Kit":
			//
			// This phase can take a while so check progress using
			// the associated Integration Kit's progress
			//
			newLogMsg = fmt.Sprintf("The building kit for integration '%s' is being initialised. This may take some time ...", integrationID)
			if integration.Status.IntegrationKit == nil {
				//
				// Not created yet so wait quietly
				//
				return false, nil
			}

			integrationKit := v1.IntegrationKit{
				TypeMeta: metav1.TypeMeta{
					Kind:       v1.IntegrationKitKind,
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: integration.Status.IntegrationKit.Namespace,
					Name:      integration.Status.IntegrationKit.Name,
				},
			}
			ikKey := k8sclient.ObjectKey{
				Namespace: integration.Status.IntegrationKit.Namespace,
				Name:      integration.Status.IntegrationKit.Name,
			}

			//
			// Query for the integration kit
			//
			if err := c.Get(o.Context, ikKey, &integrationKit); err != nil {
				if !k8errors.IsNotFound(err) {
					//
					// Not created yet so wait quietly
					//
					return false, nil
				}
				//
				// Integration kit query made an error
				//
				return false, err
			}

			//
			// Found the building kit so output its phase
			//
			newLogMsg = fmt.Sprintf("The building kit for integration '%s' is at: %s", integrationID, integrationKit.Status.Phase)
		default:
			//
			// Integration is still building, deploying or even in error
			//
			newLogMsg = fmt.Sprintf("Integration '%s' is at: %s ...", integrationID, phase)
		}

		return false, nil
	})

	if err != nil {
		return err
	}

	// Let's add a Wait point, otherwise the script terminates
	<-o.Context.Done()

	return nil
}
