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
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/olm"
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
	cmd.Flags().Bool("skip-crd", true, "Do not uninstall the Camel K Custom Resource Definitions (CRD)")
	cmd.Flags().Bool("skip-role-bindings", false, "Do not uninstall the Camel K Role Bindings in the current namespace")
	cmd.Flags().Bool("skip-roles", false, "Do not uninstall the Camel K Roles in the current namespace")
	cmd.Flags().Bool("skip-cluster-role-bindings", true, "Do not uninstall the Camel K Cluster Role Bindings")
	cmd.Flags().Bool("skip-cluster-roles", true, "Do not uninstall the Camel K Cluster Roles")
	cmd.Flags().Bool("skip-integration-platform", false,
		"Do not uninstall the Camel K Integration Platform in the current namespace")
	cmd.Flags().Bool("skip-service-accounts", false,
		"Do not uninstall the Camel K Service Accounts in the current namespace")
	cmd.Flags().Bool("skip-config-maps", false, "Do not uninstall the Camel K Config Maps in the current namespace")
	cmd.Flags().Bool("skip-registry-secret", false,
		"Do not uninstall the Camel K Registry Secret in the current namespace")
	cmd.Flags().Bool("skip-kamelets", false, "Do not uninstall the Kamelets in the current namespace")
	cmd.Flags().Bool("global", false, "Indicates that a global installation is going to be uninstalled (affects OLM)")
	cmd.Flags().Bool("olm", true, "Try to uninstall via OLM (Operator Lifecycle Manager) if available")
	cmd.Flags().String("olm-operator-name", "", "Name of the Camel K operator in the OLM source or marketplace")
	cmd.Flags().String("olm-package", "", "Name of the Camel K package in the OLM source or marketplace")
	cmd.Flags().String("olm-global-namespace", "", "A namespace containing an OperatorGroup that defines "+
		"global scope for the operator (used in combination with the --global flag)")
	cmd.Flags().Bool("all", false, "Do uninstall all Camel K resources")

	return &cmd, &options
}

type uninstallCmdOptions struct {
	*RootCmdOptions
	SkipOperator            bool `mapstructure:"skip-operator"`
	SkipCrd                 bool `mapstructure:"skip-crd"`
	SkipRoleBindings        bool `mapstructure:"skip-role-bindings"`
	SkipRoles               bool `mapstructure:"skip-roles"`
	SkipClusterRoleBindings bool `mapstructure:"skip-cluster-role-bindings"`
	SkipClusterRoles        bool `mapstructure:"skip-cluster-roles"`
	SkipIntegrationPlatform bool `mapstructure:"skip-integration-platform"`
	SkipServiceAccounts     bool `mapstructure:"skip-service-accounts"`
	SkipConfigMaps          bool `mapstructure:"skip-config-maps"`
	SkipRegistrySecret      bool `mapstructure:"skip-registry-secret"`
	SkipKamelets            bool `mapstructure:"skip-kamelets"`
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

func (o *uninstallCmdOptions) uninstall(cmd *cobra.Command, _ []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	if !o.SkipIntegrationPlatform {
		if err = o.uninstallIntegrationPlatform(o.Context, c); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Integration Platform removed from namespace %s\n", o.Namespace)
	}

	if err = o.uninstallNamespaceResources(o.Context, cmd, c); err != nil {
		return err
	}

	// nolint: ifshort
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

	if !uninstallViaOLM {
		if !o.SkipOperator {
			if err = o.uninstallOperator(o.Context, c); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Camel K Operator removed from namespace %s\n", o.Namespace)
		}

		if err = o.uninstallNamespaceRoles(o.Context, cmd, c); err != nil {
			return err
		}

		if err = o.uninstallClusterWideResources(o.Context, cmd, c, o.Namespace); err != nil {
			return err
		}

	}

	return nil
}

func (o *uninstallCmdOptions) uninstallOperator(ctx context.Context, c client.Client) error {
	api := c.AppsV1()

	deployments, err := api.Deployments(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		err := api.Deployments(o.Namespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterWideResources(ctx context.Context, cmd *cobra.Command,
	c client.Client, namespace string) error {
	if !o.SkipCrd || o.UninstallAll {
		if err := o.uninstallCrd(ctx, c); err != nil {
			if k8serrors.IsForbidden(err) {
				return createActionNotAuthorizedError(cmd)
			}
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Custom Resource Definitions removed from cluster")
	}

	if err := o.removeSubjectFromClusterRoleBindings(ctx, c, namespace); err != nil {
		if k8serrors.IsForbidden(err) {
			// Let's print a warning message and continue
			fmt.Fprintln(cmd.ErrOrStderr(),
				"Current user is not authorized to remove the operator ServiceAccount from the cluster role bindings")
		} else if err != nil {
			return err
		}
	}

	if !o.SkipClusterRoleBindings || o.UninstallAll {
		if err := o.uninstallClusterRoleBindings(ctx, c); err != nil {
			if k8serrors.IsForbidden(err) {
				return createActionNotAuthorizedError(cmd)
			}
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Cluster Role Bindings removed from cluster")
	}

	if !o.SkipClusterRoles || o.UninstallAll {
		if err := o.uninstallClusterRoles(ctx, c); err != nil {
			if k8serrors.IsForbidden(err) {
				return createActionNotAuthorizedError(cmd)
			}
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Cluster Roles removed from cluster")
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallNamespaceRoles(ctx context.Context, cmd *cobra.Command,
	c client.Client) error {
	if !o.SkipRoleBindings {
		if err := o.uninstallRoleBindings(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Role Bindings removed from namespace", o.Namespace)
	}

	if !o.SkipRoles {
		if err := o.uninstallRoles(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Roles removed from namespace", o.Namespace)
	}

	if !o.SkipServiceAccounts {
		if err := o.uninstallServiceAccounts(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Service Accounts removed from namespace", o.Namespace)
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallNamespaceResources(ctx context.Context, cmd *cobra.Command,
	c client.Client) error {
	if !o.SkipConfigMaps {
		if err := o.uninstallConfigMaps(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Config Maps removed from namespace", o.Namespace)
	}

	if !o.SkipRegistrySecret {
		if err := o.uninstallRegistrySecret(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Registry Secret removed from namespace", o.Namespace)
	}

	if !o.SkipKamelets {
		if err := o.uninstallKamelets(ctx, c); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Camel K Platform Kamelets removed from namespace", o.Namespace)
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallCrd(ctx context.Context, c client.Client) error {
	restClient, err := apiutil.RESTClientForGVK(
		schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1"}, false,
		c.GetConfig(), serializer.NewCodecFactory(c.GetScheme()))
	if err != nil {
		return err
	}

	result := restClient.
		Delete().
		Param("labelSelector", "app=camel-k").
		Resource("customresourcedefinitions").
		Do(ctx)

	if result.Error() != nil {
		return result.Error()
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRoles(ctx context.Context, c client.Client) error {
	api := c.RbacV1()

	roleBindings, err := api.Roles(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, roleBinding := range roleBindings.Items {
		err := api.Roles(o.Namespace).Delete(ctx, roleBinding.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRoleBindings(ctx context.Context, c client.Client) error {
	api := c.RbacV1()

	roleBindings, err := api.RoleBindings(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, roleBinding := range roleBindings.Items {
		err := api.RoleBindings(o.Namespace).Delete(ctx, roleBinding.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterRoles(ctx context.Context, c client.Client) error {
	api := c.RbacV1()

	clusterRoles, err := api.ClusterRoles().List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, clusterRole := range clusterRoles.Items {
		err := api.ClusterRoles().Delete(ctx, clusterRole.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) removeSubjectFromClusterRoleBindings(ctx context.Context, c client.Client,
	namespace string) error {
	api := c.RbacV1()

	clusterRoleBindings, err := api.ClusterRoleBindings().List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	// Remove the subject corresponding to this operator install
	for crbIndex, clusterRoleBinding := range clusterRoleBindings.Items {
		for i, subject := range clusterRoleBinding.Subjects {
			if subject.Name == "camel-k-operator" && subject.Namespace == namespace {
				clusterRoleBinding.Subjects =
					append(clusterRoleBinding.Subjects[:i], clusterRoleBinding.Subjects[i+1:]...)
				_, err = api.ClusterRoleBindings().
					Update(ctx, &clusterRoleBindings.Items[crbIndex], metav1.UpdateOptions{})
				if err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallClusterRoleBindings(ctx context.Context, c client.Client) error {
	api := c.RbacV1()

	clusterRoleBindings, err := api.ClusterRoleBindings().List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, clusterRoleBinding := range clusterRoleBindings.Items {
		err := api.ClusterRoleBindings().Delete(ctx, clusterRoleBinding.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallServiceAccounts(ctx context.Context, c client.Client) error {
	api := c.CoreV1()

	serviceAccountList, err := api.ServiceAccounts(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, serviceAccount := range serviceAccountList.Items {
		err := api.ServiceAccounts(o.Namespace).Delete(ctx, serviceAccount.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallIntegrationPlatform(ctx context.Context, c client.Client) error {
	integrationPlatforms, err := c.CamelV1().IntegrationPlatforms(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, integrationPlatform := range integrationPlatforms.Items {
		err := c.CamelV1().IntegrationPlatforms(o.Namespace).
			Delete(ctx, integrationPlatform.GetName(), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallConfigMaps(ctx context.Context, c client.Client) error {
	api := c.CoreV1()

	configMapsList, err := api.ConfigMaps(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, configMap := range configMapsList.Items {
		err := api.ConfigMaps(o.Namespace).Delete(ctx, configMap.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallRegistrySecret(ctx context.Context, c client.Client) error {
	api := c.CoreV1()

	secretsList, err := api.Secrets(o.Namespace).List(ctx, defaultListOptions)
	if err != nil {
		return err
	}

	for _, secret := range secretsList.Items {
		err := api.Secrets(o.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *uninstallCmdOptions) uninstallKamelets(ctx context.Context, c client.Client) error {
	kameletList := v1alpha1.NewKameletList()
	if err := c.List(ctx, &kameletList, ctrl.InNamespace(o.Namespace)); err != nil {
		return err
	}

	for i := range kameletList.Items {
		// remove only platform Kamelets (user-defined Kamelets should be skipped)
		if kameletList.Items[i].Labels[v1alpha1.KameletBundledLabel] == "true" {
			err := c.Delete(ctx, &kameletList.Items[i])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createActionNotAuthorizedError(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.ErrOrStderr(),
		"Current user is not authorized to remove cluster-wide objects like custom resource definitions or cluster roles")
	// nolint: lll
	msg := `login as cluster-admin and execute "kamel uninstall" or use flags "--skip-crd --skip-cluster-roles --skip-cluster-role-bindings"`
	return errors.New(msg)
}
