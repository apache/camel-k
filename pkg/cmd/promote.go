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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	util "github.com/apache/camel-k/v2/pkg/util/gitops"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdPromote --.
func newCmdPromote(rootCmdOptions *RootCmdOptions) (*cobra.Command, *promoteCmdOptions) {
	options := promoteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "promote my-it [--to <namespace>] [-x <promoted-operator-id>]",
		Short:   "Promote an Integration/Pipe from an environment to another",
		Long:    "Promote an Integration/Pipe from an environment to another, for example from a Development environment to a Production environment",
		PreRunE: decode(&options, options.Flags),
		RunE:    options.run,
	}

	cmd.Flags().String("to", "", "The namespace where to promote the Integration/Pipe")
	cmd.Flags().StringP("to-operator", "x", "", "The operator id which will reconcile the promoted Integration/Pipe")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().BoolP("image", "i", false, "Output the container image only")
	cmd.Flags().String("export-gitops-dir", "", "Export to a Kustomize GitOps overlay structure")
	cmd.Flags().Bool("overwrite", false, "Overwrite the overlay if it exists")

	return &cmd, &options
}

type promoteCmdOptions struct {
	*RootCmdOptions

	To           string `mapstructure:"to"                yaml:",omitempty"`
	ToOperator   string `mapstructure:"to-operator"       yaml:",omitempty"`
	OutputFormat string `mapstructure:"output"            yaml:",omitempty"`
	Image        bool   `mapstructure:"image"             yaml:",omitempty"`
	ToGitOpsDir  string `mapstructure:"export-gitops-dir" yaml:",omitempty"`
	Overwrite    bool   `mapstructure:"overwrite"         yaml:",omitempty"`
}

func (o *promoteCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("promote requires an Integration/Pipe name argument")
	}
	if o.To == "" {
		return errors.New("promote requires a destination namespace as --to argument")
	}
	if o.To == o.Namespace {
		return errors.New("source and destination namespaces must be different in order to avoid promoted Integration/Pipe " +
			"clashes with the source Integration/Pipe")
	}

	return nil
}

func (o *promoteCmdOptions) run(cmd *cobra.Command, args []string) error {
	if err := o.validate(cmd, args); err != nil {
		return err
	}

	name := args[0]
	c, err := o.GetCmdClient()
	if err != nil {
		return fmt.Errorf("could not retrieve cluster client: %w", err)
	}
	if !o.isDryRun() {
		// Skip these checks if in dry mode
		opSource, err := operatorInfo(o.Context, c, o.Namespace)
		if err != nil {
			return fmt.Errorf("could not retrieve info for Camel K operator source: %w", err)
		}
		opDest, err := operatorInfo(o.Context, c, o.To)
		if err != nil {
			return fmt.Errorf("could not retrieve info for Camel K operator destination: %w", err)
		}

		err = checkOpsCompatibility(cmd, opSource, opDest)
		if err != nil {
			return fmt.Errorf("could not verify operators compatibility: %w", err)
		}
	}

	promotePipe := false
	var sourceIntegration *v1.Integration
	// We first look if a Pipe with the name exists
	sourcePipe, err := o.getPipe(c, name)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("problems looking for Pipe "+name+": %w", err)
	}
	if sourcePipe != nil {
		promotePipe = true
	}
	sourceIntegration, err = getIntegration(o.Context, c, name, o.Namespace)
	if err != nil {
		return fmt.Errorf("could not get Integration "+name+": %w", err)
	}
	if sourceIntegration.Status.Phase != v1.IntegrationPhaseRunning &&
		sourceIntegration.Status.Phase != v1.IntegrationPhaseBuildComplete {
		return fmt.Errorf("could not promote an Integration in %s status", sourceIntegration.Status.Phase)
	}
	sourceKit, err := o.getIntegrationKit(c, sourceIntegration.Status.IntegrationKit)
	if err != nil {
		return err
	}
	// Image only mode
	if o.Image {
		showImageOnly(cmd, sourceIntegration)

		return nil
	}

	// Pipe promotion
	if promotePipe {
		destPipe := util.EditPipe(sourcePipe, sourceIntegration, sourceKit, o.To, o.ToOperator)
		if o.OutputFormat != "" {
			return showPipeOutput(cmd, destPipe, o.OutputFormat, c.GetScheme())
		}
		if o.ToGitOpsDir != "" {
			err = util.AppendKustomizePipe(destPipe, o.ToGitOpsDir, o.Overwrite)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), `Exported a Kustomize based Gitops directory to `+o.ToGitOpsDir+` for "`+name+`" Pipe`)

			return nil
		}
		replaced, err := o.replaceResource(destPipe)
		if err != nil {
			return err
		}
		if !replaced {
			fmt.Fprintln(cmd.OutOrStdout(), `Promoted Pipe "`+name+`" created`)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), `Promoted Pipe "`+name+`" updated`)
		}

		return nil
	}

	// Plain Integration promotion
	destIntegration := util.EditIntegration(sourceIntegration, sourceKit, o.To, o.ToOperator)
	if o.OutputFormat != "" {
		return showIntegrationOutput(cmd, destIntegration, o.OutputFormat)
	}
	if o.ToGitOpsDir != "" {
		err = util.AppendKustomizeIntegration(destIntegration, o.ToGitOpsDir, o.Overwrite)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), `Exported a Kustomize based Gitops directory to `+o.ToGitOpsDir+` for "`+name+`" Integration`)

		return nil
	}

	replaced, err := o.replaceResource(destIntegration)
	if err != nil {
		return err
	}
	if !replaced {
		fmt.Fprintln(cmd.OutOrStdout(), `Promoted Integration "`+name+`" created`)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), `Promoted Integration "`+name+`" updated`)
	}

	return nil
}

func checkOpsCompatibility(cmd *cobra.Command, source, dest map[string]string) error {
	if !compatibleVersions(source["Version"], dest["Version"], cmd) {
		return fmt.Errorf("source (%s) and destination (%s) Camel K operator versions are not compatible", source["Version"], dest["Version"])
	}
	if !compatibleVersions(source["Runtime Version"], dest["Runtime Version"], cmd) {
		return fmt.Errorf("source (%s) and destination (%s) Camel K runtime versions are not compatible", source["Runtime Version"], dest["Runtime Version"])
	}
	if source["Registry Address"] != dest["Registry Address"] {
		return fmt.Errorf("source (%s) and destination (%s) Camel K container images registries are not the same", source["Registry Address"], dest["Registry Address"])
	}

	return nil
}

func (o *promoteCmdOptions) getPipe(c client.Client, name string) (*v1.Pipe, error) {
	it := v1.NewPipe(o.Namespace, name)
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}
	if err := c.Get(o.Context, key, &it); err != nil {
		return nil, err
	}

	return &it, nil
}

func (o *promoteCmdOptions) getIntegrationKit(c client.Client, ref *corev1.ObjectReference) (*v1.IntegrationKit, error) {
	if ref == nil {
		return nil, nil
	}

	return kubernetes.GetIntegrationKit(o.Context, c, ref.Name, ref.Namespace)
}

func (o *promoteCmdOptions) replaceResource(res k8sclient.Object) (bool, error) {
	return kubernetes.ReplaceResource(o.Context, o._client, res)
}

func (o *promoteCmdOptions) isDryRun() bool {
	return o.OutputFormat != "" || o.Image
}

func showImageOnly(cmd *cobra.Command, integration *v1.Integration) {
	fmt.Fprintln(cmd.OutOrStdout(), integration.Status.Image)
}
