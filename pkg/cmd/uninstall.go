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
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"

	"github.com/apache/camel-k/pkg/client"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCmdUninstall(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := uninstallCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Camel K from a Kubernetes cluster",
		Long:  `Uninstalls Camel K from a Kubernetes or OpenShift cluster.`,
		Run:   options.uninstall,
	}

	cmd.Flags().BoolVar(&options.skipOperator, "skip-operator", false, "Do not uninstall the Camel-K Operator in the current namespace")
	cmd.Flags().BoolVar(&options.skipCrd, "skip-crd", false, "Do not uninstall the Camel-k Custom Resource Definitions (CRD) in the current namespace")
	cmd.Flags().BoolVar(&options.skipRoleBindings, "skip-role-bindings", false, "Do not uninstall the Camel-K Role Bindings in the current namespace")
	cmd.Flags().BoolVar(&options.skipRoles, "skip-roles", false, "Do not uninstall the Camel-K Roles in the current namespace")
	cmd.Flags().BoolVar(&options.skipClusterRoles, "skip-cluster-roles", false, "Do not uninstall the Camel-K Cluster Roles in the current namespace")
	cmd.Flags().BoolVar(&options.skipIntegrationPlatform, "skip-Integration-Platform", false, "Do not uninstall the Camel-K Integration Platform in the current namespace")

	// completion support
	configureBashAnnotationForFlag(
		&cmd,
		"context",
		map[string][]string{
			cobra.BashCompCustom: {"kamel_kubectl_get_known_integrationcontexts"},
		},
	)

	return &cmd
}

type uninstallCmdOptions struct {
	*RootCmdOptions
	skipOperator            bool
	skipCrd                 bool
	skipRoleBindings        bool
	skipRoles               bool
	skipClusterRoles        bool
	skipIntegrationPlatform bool
}

func (o *uninstallCmdOptions) uninstall(_ *cobra.Command, _ []string) {
	c, err := o.GetCmdClient()
	if err != nil {
		fmt.Print(err)
		return
	}

	if !o.skipIntegrationPlatform{
		if err = o.uninstallIntegrationPlatform(c); err != nil {
			fmt.Print(err)
			return
		}
		fmt.Printf("Camel-K Integration Platform removed from namespace %s\n", o.Namespace)
	}

	if err = o.uninstallClusterWideResources(c); err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("Camel-K Cluster Wide Resources removed from namespace %s\n", o.Namespace)

	if !o.skipOperator {
		if err = o.uninstallOperator(c); err != nil {
			fmt.Print(err)
			return
		}
		fmt.Printf("Camel-K Operator removed from namespace %s\n", o.Namespace)
	}

	return
}

func (o *uninstallCmdOptions) uninstallOperator(c client.Client) error {
	api := c.AppsV1()

	listOptions := metav1.ListOptions{
		LabelSelector: "app=camel-k,camel.apache.org/component=operator",
	}

	deployments, err := api.Deployments(o.Namespace).List(listOptions)
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		api.Deployments(o.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterWideResources(c client.Client) error {
	if !o.skipCrd {
		if err := o.uninstallCrd(c); err != nil {
			return err
		}
		fmt.Printf("Camel-K Custom Resourse Definitions removed from namespace %s\n", o.Namespace)
	}

	if !o.skipRoleBindings {
		if err := o.uninstallRoleBindings(c); err != nil {
			return err
		}
		fmt.Printf("Camel-K Role Bindings removed from namespace %s\n", o.Namespace)
	}

	if !o.skipClusterRoles {
		if err := o.uninstallClusterRoles(c); err != nil {
			return err
		}
		fmt.Printf("Camel-K Cluster Roles removed from namespace %s\n", o.Namespace)
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallCrd(c client.Client) error {
	restClient, err := customclient.GetClientFor(c, "apiextensions.k8s.io", "v1beta1")
	if err != nil {
		return err
	}

	result := restClient.
		Delete().
		Param("labelSelector", "app=camel-k").
		Resource("customresourcedefinitions").
		Do()

	if result.Error() != nil && !k8serrors.IsAlreadyExists(result.Error()) {
		return result.Error()
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRoleBindings(c client.Client) error {
	api := c.RbacV1()

	listOptions := metav1.ListOptions{
		LabelSelector: "app=camel-k",
	}

	roleBindings, err := api.RoleBindings(o.Namespace).List(listOptions)
	if err != nil {
		return err
	}

	for _, roleBinding := range roleBindings.Items {
		api.ClusterRoles().Delete(roleBinding.Name, &metav1.DeleteOptions{})
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterRoles(c client.Client) error {
	api := c.RbacV1()

	listOptions := metav1.ListOptions{
		LabelSelector: "app=camel-k",
	}

	clusterRoles, err := api.ClusterRoles().List(listOptions)
	if err != nil {
		return err
	}

	for _, clusterRole := range clusterRoles.Items {
		api.ClusterRoles().Delete(clusterRole.Name, &metav1.DeleteOptions{})
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallIntegrationPlatform(c client.Client) error {
	api, err := customclient.GetDefaultDynamicClientFor("integrations", o.Namespace)
	if err != nil {
		return err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: "app=camel-k,camel.apache.org/component=operator",
	}

	integrationPlatforms, err  := api.List(listOptions)
	if err != nil {
		return err
	}

	for _, integrationPlatform := range integrationPlatforms.Items {
		api.Delete(integrationPlatform.GetName(), &metav1.DeleteOptions{})
	}

	return nil
}