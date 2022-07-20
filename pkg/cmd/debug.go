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
	"os"
	"os/signal"
	"syscall"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	camelv1 "github.com/apache/camel-k/pkg/client/camel/clientset/versioned/typed/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	k8slog "github.com/apache/camel-k/pkg/util/kubernetes/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func newCmdDebug(rootCmdOptions *RootCmdOptions) (*cobra.Command, *debugCmdOptions) {
	options := debugCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "debug [integration name]",
		Short: "Debug an integration running on Kubernetes",
		Long: "Set an integration running on the Kubernetes cluster in debug mode and forward ports " +
			"in order to connect a remote debugger running on the local host.",
		Args:    options.validateArgs,
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	cmd.Flags().Bool("suspend", true, "Suspend the integration on startup, to let the debugger attach from the beginning")
	cmd.Flags().Uint("port", 5005, "Local port to use for port-forwarding")
	cmd.Flags().Uint("remote-port", 5005, "Remote port to use for port-forwarding")

	// completion support
	configureKnownCompletions(&cmd)

	return &cmd, &options
}

type debugCmdOptions struct {
	*RootCmdOptions `json:"-"`
	Suspend         bool `mapstructure:"suspend" yaml:",omitempty"`
	Port            uint `mapstructure:"port" yaml:",omitempty"`
	RemotePort      uint `mapstructure:"remote-port" yaml:",omitempty"`
}

func (o *debugCmdOptions) validateArgs(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("run expects 1 argument, received 0")
	}
	return nil
}

func (o *debugCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCamelCmdClient()
	if err != nil {
		return err
	}

	name := args[0]

	it, err := c.Integrations(o.Namespace).Get(o.Context, name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return fmt.Errorf("integration %q not found in namespace %q", name, o.Namespace)
	} else if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Enabling debug mode on integration %q...\n", name)
	if _, err := o.toggleDebug(c, it, true); err != nil {
		return err
	}

	cs := make(chan os.Signal, 1)
	signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cs
		if o.Context.Err() != nil {
			// Context canceled
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), `Disabling debug mode on integration "`+name+`"`)
		it, err := c.Integrations(o.Namespace).Get(o.Context, name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
			os.Exit(1)
		}
		_, err = o.toggleDebug(c, it, false)
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}()

	cmdClient, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	selector := fmt.Sprintf("camel.apache.org/debug=true,camel.apache.org/integration=%s", name)

	go func() {
		err = k8slog.PrintUsingSelector(o.Context, cmd, cmdClient, o.Namespace, "integration", selector, cmd.OutOrStdout())
		if err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
		}
	}()

	return kubernetes.PortForward(o.Context, cmdClient, o.Namespace, selector, o.Port, o.RemotePort,
		cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func (o *debugCmdOptions) toggleDebug(c camelv1.IntegrationsGetter, it *v1.Integration, active bool) (
	*v1.Integration, error,
) {
	if it.Spec.Traits.JVM == nil {
		it.Spec.Traits.JVM = &traitv1.JVMTrait{}
	}
	jvmTrait := it.Spec.Traits.JVM

	if active {
		jvmTrait.Debug = pointer.Bool(true)
		jvmTrait.DebugSuspend = pointer.Bool(o.Suspend)
	} else {
		jvmTrait.Debug = nil
		jvmTrait.DebugSuspend = nil
	}

	return c.Integrations(it.Namespace).Update(o.Context, it, metav1.UpdateOptions{})
}
