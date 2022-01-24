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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/apache/camel-k/pkg/util"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func newCmdDump(rootCmdOptions *RootCmdOptions) (*cobra.Command, *dumpCmdOptions) {
	options := dumpCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "dump [filename]",
		Short:   "Dump the state of namespace",
		Long:    `Dump the state of currently used namespace. If no filename will be specified, the output will be on stdout`,
		PreRunE: decode(&options),
		RunE:    options.dump,
	}

	cmd.Flags().Int("logLines", 100, "Number of log lines to dump")
	return &cmd, &options
}

type dumpCmdOptions struct {
	*RootCmdOptions
	LogLines int `mapstructure:"logLines"`
}

func (o *dumpCmdOptions) dump(cmd *cobra.Command, args []string) (err error) {
	c, err := o.GetCmdClient()
	if err != nil {
		return
	}

	if len(args) == 1 {
		err = util.WithFile(args[0], os.O_RDWR|os.O_CREATE, 0o644, func(file *os.File) error {
			return dumpNamespace(o.Context, c, o.Namespace, file, o.LogLines)
		})
	} else {
		err = dumpNamespace(o.Context, c, o.Namespace, cmd.OutOrStdout(), o.LogLines)
	}

	return
}

func dumpNamespace(ctx context.Context, c client.Client, ns string, out io.Writer, logLines int) error {
	camelClient, err := versioned.NewForConfig(c.GetConfig())
	if err != nil {
		return err
	}
	pls, err := camelClient.CamelV1().IntegrationPlatforms(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d platforms:\n", len(pls.Items))
	for _, p := range pls.Items {
		ref := p
		pdata, err := kubernetes.ToYAML(&ref)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "---\n%s\n---\n", string(pdata))
	}

	its, err := camelClient.CamelV1().Integrations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d integrations:\n", len(its.Items))
	for _, integration := range its.Items {
		ref := integration
		pdata, err := kubernetes.ToYAML(&ref)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "---\n%s\n---\n", string(pdata))
	}

	iks, err := camelClient.CamelV1().IntegrationKits(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d integration kits:\n", len(iks.Items))
	for _, ik := range iks.Items {
		ref := ik
		pdata, err := kubernetes.ToYAML(&ref)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "---\n%s\n---\n", string(pdata))
	}

	cms, err := c.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d ConfigMaps:\n", len(cms.Items))
	for _, cm := range cms.Items {
		ref := cm
		pdata, err := kubernetes.ToYAML(&ref)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "---\n%s\n---\n", string(pdata))
	}

	deployments, err := c.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d deployments:\n", len(deployments.Items))
	for _, deployment := range deployments.Items {
		ref := deployment
		data, err := kubernetes.ToYAML(&ref)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "---\n%s\n---\n", string(data))
	}

	lst, err := c.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\nFound %d pods:\n", len(lst.Items))
	for _, pod := range lst.Items {
		fmt.Fprintf(out, "name=%s\n", pod.Name)
		dumpConditions("  ", pod.Status.Conditions, out)
		fmt.Fprintf(out, "  logs:\n")
		var allContainers []v1.Container
		allContainers = append(allContainers, pod.Spec.InitContainers...)
		allContainers = append(allContainers, pod.Spec.Containers...)
		for _, container := range allContainers {
			pad := "    "
			fmt.Fprintf(out, "%s%s\n", pad, container.Name)
			err := dumpLogs(ctx, c, fmt.Sprintf("%s> ", pad), ns, pod.Name, container.Name, out, logLines)
			if err != nil {
				fmt.Fprintf(out, "%sERROR while reading the logs: %v\n", pad, err)
			}
		}
	}
	return nil
}

func dumpConditions(prefix string, conditions []v1.PodCondition, out io.Writer) {
	for _, cond := range conditions {
		fmt.Fprintf(out, "%scondition type=%s, status=%s, reason=%s, message=%q\n", prefix, cond.Type, cond.Status, cond.Reason, cond.Message)
	}
}

func dumpLogs(ctx context.Context, c client.Client, prefix string, ns string, name string, container string, out io.Writer, logLines int) error {
	lines := int64(logLines)
	stream, err := c.CoreV1().Pods(ns).GetLogs(name, &v1.PodLogOptions{
		Container: container,
		TailLines: &lines,
	}).Stream(ctx)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stream)
	printed := false
	for scanner.Scan() {
		printed = true
		fmt.Fprintf(out, "%s%s\n", prefix, scanner.Text())
	}
	if !printed {
		fmt.Fprintf(out, "%s[no logs available]\n", prefix)
	}
	return stream.Close()
}
