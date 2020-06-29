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

	"github.com/spf13/viper"

	"github.com/apache/camel-k/pkg/util/olm"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCmdUninstall(rootCmdOptions *RootCmdOptions) (*cobra.Command, *uninstallCmdOptions) {
	options := uninstallCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall Camel K from a Kubernetes cluster",
		Long:    `Uninstalls Camel K from a Kubernetes or OpenShift cluster.`,
		PreRunE: options.decode,
		RunE:    options.uninstall,
	}

	cmd.Flags().Bool("skip-operator", false, "Do not uninstall the Camel K Operator in the current namespace")
	cmd.Flags().Bool("skip-crd", true, "Do not uninstall the Camel-k Custom Resource Definitions (CRD)")
	cmd.Flags().Bool("skip-role-bindings", false, "Do not uninstall the Camel K Role Bindings in the current namespace")
	cmd.Flags().Bool("skip-roles", false, "Do not uninstall the Camel K Roles in the current namespace")
	cmd.Flags().Bool("skip-cluster-roles", true, "Do not uninstall the Camel K Cluster Roles")
	cmd.Flags().Bool("skip-integration-platform", false, "Do not uninstall the Camel K Integration Platform in the current namespace")
	cmd.Flags().Bool("skip-service-accounts", false, "Do not uninstall the Camel K Service Accounts in the current namespace")
	cmd.Flags().Bool("skip-config-maps", false, "Do not uninstall the Camel K Config Maps in the current namespace")
	cmd.Flags().Bool("global", false, "Indicates that a global installation is going to be uninstalled (affects OLM)")
	cmd.Flags().Bool("olm", true, "Try to uninstall via OLM (Operator Lifecycle Manager) if available")
	cmd.Flags().String("olm-operator-name", olm.DefaultOperatorName, "Name of the Camel K operator in the OLM source or marketplace")
	cmd.Flags().String("olm-package", olm.DefaultPackage, "Name of the Camel K package in the OLM source or marketplace")
	cmd.Flags().String("olm-global-namespace", olm.DefaultGlobalNamespace, "A namespace containing an OperatorGroup that defines "+
		"global scope for the operator (used in combination with the --global flag)")
	cmd.Flags().Bool("all", false, "Do uninstall all Camel-K resources")

	return &cmd, &options
}

type uninstallCmdOptions struct {
	*RootCmdOptions
	SkipOperator            bool `mapstructure:"skip-operator"`
	SkipCrd                 bool `mapstructure:"skip-crd"`
	SkipRoleBindings        bool `mapstructure:"skip-role-bindings"`
	SkipRoles               bool `mapstructure:"skip-roles"`
	SkipClusterRoles        bool `mapstructure:"skip-cluster-roles"`
	SkipIntegrationPlatform bool `mapstructure:"skip-integration-platform"`
	SkipServiceAccounts     bool `mapstructure:"skip-service-accounts"`
	SkipConfigMaps          bool `mapstructure:"skip-config-maps"`
	Global                  bool `mapstructure:"global"`
	OlmEnabled              bool `mapstructure:"olm"`
	UninstallAll            bool `mapstructure:"all"`

	OlmOptions olm.Options
}

var defaultListOptions = metav1.ListOptions{
	LabelSelector: "app=camel-k",
}

func (o *uninstallCmdOptions) decode(cmd *cobra.Command, _ []string) error {
	path := pathToRoot(cmd)
	if err := decodeKey(o, path); err != nil {
		return err
	}

	o.OlmOptions.OperatorName = viper.GetString(path + ".olm-operator-name")
	o.OlmOptions.Package = viper.GetString(path + ".olm-package")
	o.OlmOptions.GlobalNamespace = viper.GetString(path + ".olm-global-namespace")

	return nil
}

// nolint: gocyclo
func (o *uninstallCmdOptions) uninstall(cmd *cobra.Command, _ []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	uninstallViaOLM := false
	if o.OlmEnabled {
		var err error
		if uninstallViaOLM, err = olm.IsAPIAvailable(o.Context, c, o.Namespace); err != nil {
			return errors.Wrap(err, "error while checking OLM availability. Run with '--olm=false' to skip this check")
		}

		if uninstallViaOLM {
			fmt.Fprintln(cmd.OutOrStdout(), "OLM is available in the cluster")
			if err = olm.Uninstall(o.Context, c, o.Namespace, o.Global, o.OlmOptions); err != nil {
				return err
			}
			where := fmt.Sprintf("from namespace %s", o.Namespace)
			if o.Global {
				where = "globally"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Camel K OLM service removed %s\n", where)
		}
	}

	if !o.SkipIntegrationPlatform {
		if err = o.uninstallIntegrationPlatform(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Integration Platform removed from namespace %s\n", o.Namespace)
	}

	if err = o.uninstallNamespaceResources(c); err != nil {
		return err
	}

	if !uninstallViaOLM {
		if !o.SkipOperator {
			if err = o.uninstallOperator(c); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Camel K Operator removed from namespace %s\n", o.Namespace)
		}

		if err = o.uninstallNamespaceRoles(c); err != nil {
			return err
		}

		if err = o.uninstallClusterWideResources(c); err != nil {
			return err
		}

	}

	return nil
}

func (o *uninstallCmdOptions) uninstallOperator(c client.Client) error {
	api := c.AppsV1()

	deployments, err := api.Deployments(o.Namespace).List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		err := api.Deployments(o.Namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterWideResources(c client.Client) error {
	if !o.SkipCrd || o.UninstallAll {
		if err := o.uninstallCrd(c); err != nil {
			if k8serrors.IsForbidden(err) {
				return createActionNotAuthorizedError()
			}
			return err
		}
		fmt.Printf("Camel K Custom Resource Definitions removed from cluster\n")
	}

	if !o.SkipClusterRoles || o.UninstallAll {
		if err := o.uninstallClusterRoles(c); err != nil {
			if k8serrors.IsForbidden(err) {
				return createActionNotAuthorizedError()
			}
			return err
		}
		fmt.Printf("Camel K Cluster Roles removed from cluster\n")
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallNamespaceRoles(c client.Client) error {
	if !o.SkipRoleBindings {
		if err := o.uninstallRoleBindings(c); err != nil {
			return err
		}
		fmt.Printf("Camel K Role Bindings removed from namespace %s\n", o.Namespace)
	}

	if !o.SkipRoles {
		if err := o.uninstallRoles(c); err != nil {
			return err
		}
		fmt.Printf("Camel K Roles removed from namespace %s\n", o.Namespace)
	}

	if !o.SkipServiceAccounts {
		if err := o.uninstallServiceAccounts(c); err != nil {
			return err
		}
		fmt.Printf("Camel K Service Accounts removed from namespace %s\n", o.Namespace)
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallNamespaceResources(c client.Client) error {
	if !o.SkipConfigMaps {
		if err := o.uninstallConfigMaps(c); err != nil {
			return err
		}
		fmt.Printf("Camel K Config Maps removed from namespace %s\n", o.Namespace)
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallCrd(c kubernetes.Interface) error {
	restClient, err := customclient.GetClientFor(c, "apiextensions.k8s.io", "v1")
	if err != nil {
		return err
	}

	result := restClient.
		Delete().
		Param("labelSelector", "app=camel-k").
		Resource("customresourcedefinitions").
		Do()

	if result.Error() != nil {
		return result.Error()
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRoles(c client.Client) error {
	api := c.RbacV1()

	roleBindings, err := api.Roles(o.Namespace).List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, roleBinding := range roleBindings.Items {
		err := api.Roles(o.Namespace).Delete(roleBinding.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRoleBindings(c client.Client) error {
	api := c.RbacV1()

	roleBindings, err := api.RoleBindings(o.Namespace).List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, roleBinding := range roleBindings.Items {
		err := api.RoleBindings(o.Namespace).Delete(roleBinding.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterRoles(c client.Client) error {
	api := c.RbacV1()

	clusterRoles, err := api.ClusterRoles().List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, clusterRole := range clusterRoles.Items {
		err := api.ClusterRoles().Delete(clusterRole.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallServiceAccounts(c client.Client) error {
	api := c.CoreV1()

	serviceAccountList, err := api.ServiceAccounts(o.Namespace).List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, serviceAccount := range serviceAccountList.Items {
		err := api.ServiceAccounts(o.Namespace).Delete(serviceAccount.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallIntegrationPlatform() error {
	api, err := customclient.GetDefaultDynamicClientFor("integrationplatforms", o.Namespace)
	if err != nil {
		return err
	}

	integrationPlatforms, err := api.List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, integrationPlatform := range integrationPlatforms.Items {
		err := api.Delete(integrationPlatform.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallConfigMaps(c client.Client) error {
	api := c.CoreV1()

	configMapsList, err := api.ConfigMaps(o.Namespace).List(defaultListOptions)
	if err != nil {
		return err
	}

	for _, configMap := range configMapsList.Items {
		err := api.ConfigMaps(o.Namespace).Delete(configMap.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func createActionNotAuthorizedError() error {
	fmt.Println("Current user is not authorized to remove cluster-wide objects like custom resource definitions or cluster roles")
	msg := `login as cluster-admin and execute "kamel uninstall" or use flags "--skip-crd --skip-cluster-roles"`
	return errors.New(msg)
}
