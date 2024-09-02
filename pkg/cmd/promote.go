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
	"errors"
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	return &cmd, &options
}

type promoteCmdOptions struct {
	*RootCmdOptions
	To           string `mapstructure:"to" yaml:",omitempty"`
	ToOperator   string `mapstructure:"to-operator" yaml:",omitempty"`
	OutputFormat string `mapstructure:"output" yaml:",omitempty"`
	Image        bool   `mapstructure:"image" yaml:",omitempty"`
}

func (o *promoteCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("promote requires an Integration/Pipe name argument")
	}
	if o.To == "" {
		return errors.New("promote requires a destination namespace as --to argument")
	}
	if o.To == o.Namespace {
		return errors.New("source and destination namespaces must be different in order to avoid promoted Integration/Pipe clashes with the source Integration/Pipe")
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
	sourceIntegration, err = o.getIntegration(c, name)
	if err != nil {
		return fmt.Errorf("could not get Integration "+name+": %w", err)
	}
	if sourceIntegration.Status.Phase != v1.IntegrationPhaseRunning {
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
		destPipe, destKit := o.editPipe(sourcePipe, sourceIntegration, sourceKit)
		if o.OutputFormat != "" {
			if err := showIntegrationKitOutput(cmd, destKit, o.OutputFormat); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), `---`)
			return showPipeOutput(cmd, destPipe, o.OutputFormat, c.GetScheme())
		}
		// Ensure the destination namespace has access to the source namespace images
		err = addSystemPullerRoleBinding(o.Context, c, sourceIntegration.Namespace, destPipe.Namespace)
		if err != nil {
			return err
		}
		_, err = o.replaceResource(destKit)
		if err != nil {
			return err
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
	destIntegration, destKit := o.editIntegration(sourceIntegration, sourceKit)
	if o.OutputFormat != "" {
		if err := showIntegrationKitOutput(cmd, destKit, o.OutputFormat); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), `---`)
		return showIntegrationOutput(cmd, destIntegration, o.OutputFormat)
	}
	// Ensure the destination namespace has access to the source namespace images
	err = addSystemPullerRoleBinding(o.Context, c, sourceIntegration.Namespace, destIntegration.Namespace)
	if err != nil {
		return err
	}
	_, err = o.replaceResource(destKit)
	if err != nil {
		return err
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

func (o *promoteCmdOptions) getIntegration(c client.Client, name string) (*v1.Integration, error) {
	it := v1.NewIntegration(o.Namespace, name)
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
	ik := v1.NewIntegrationKit(ref.Namespace, ref.Name)
	key := k8sclient.ObjectKey{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}
	if err := c.Get(o.Context, key, ik); err != nil {
		return nil, err
	}

	return ik, nil
}

func (o *promoteCmdOptions) editIntegration(it *v1.Integration, kit *v1.IntegrationKit) (*v1.Integration, *v1.IntegrationKit) {
	contImage := it.Status.Image
	// IntegrationKit
	dstKit := v1.NewIntegrationKit(o.To, kit.Name)
	dstKit.Spec = *kit.Spec.DeepCopy()
	dstKit.Annotations = cloneAnnotations(kit.Annotations, o.ToOperator)
	dstKit.Labels = cloneLabels(kit.Labels)
	dstKit.Labels = alterKitLabels(dstKit.Labels, kit)
	dstKit.Spec.Image = contImage
	// Integration
	dstIt := v1.NewIntegration(o.To, it.Name)
	dstIt.Spec = *it.Spec.DeepCopy()
	dstIt.Annotations = cloneAnnotations(it.Annotations, o.ToOperator)
	dstIt.Labels = cloneLabels(it.Labels)
	dstIt.Spec.IntegrationKit = &corev1.ObjectReference{
		Namespace: dstKit.Namespace,
		Name:      dstKit.Name,
		Kind:      dstKit.Kind,
	}
	// We must provide the classpath expected for the IntegrationKit. This is calculated dynamically and
	// would get lost when creating the promoted IntegrationKit (which is in .status.artifacts). For this reason
	// we must report it in the promoted Integration.
	kitClasspath := kit.Status.GetDependenciesPaths()
	if len(kitClasspath) > 0 {
		if dstIt.Spec.Traits.JVM == nil {
			dstIt.Spec.Traits.JVM = &traitv1.JVMTrait{}
		}
		jvmTrait := dstIt.Spec.Traits.JVM
		if jvmTrait.Classpath != "" {
			jvmTrait.Classpath += ":"
		}
		jvmTrait.Classpath += strings.Join(kitClasspath, ":")
	}

	return &dstIt, dstKit
}

// Return all annotations overriding the operator Id if provided.
func cloneAnnotations(ann map[string]string, operatorID string) map[string]string {
	operatorIDAnnotationSet := false
	newMap := make(map[string]string)
	for k, v := range ann {
		if k == v1.OperatorIDAnnotation {
			if operatorID != "" {
				newMap[v1.OperatorIDAnnotation] = operatorID
				operatorIDAnnotationSet = true
			}
		} else {
			newMap[k] = v
		}
	}
	if !operatorIDAnnotationSet && operatorID != "" {
		newMap[v1.OperatorIDAnnotation] = operatorID
	}
	return newMap
}

// Return all labels. The method is a reference if in the future we need to apply any filtering.
func cloneLabels(lbs map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range lbs {
		newMap[k] = v
	}
	return newMap
}

// Change labels expected by Integration Kit replacing the creator to reflect the
// fact the new kit was cloned by another one instead.
func alterKitLabels(lbs map[string]string, kit *v1.IntegrationKit) map[string]string {
	lbs[v1.IntegrationKitTypeLabel] = v1.IntegrationKitTypeExternal
	if lbs[kubernetes.CamelCreatorLabelKind] != "" {
		delete(lbs, kubernetes.CamelCreatorLabelKind)
	}
	if lbs[kubernetes.CamelCreatorLabelName] != "" {
		delete(lbs, kubernetes.CamelCreatorLabelName)
	}
	if lbs[kubernetes.CamelCreatorLabelNamespace] != "" {
		delete(lbs, kubernetes.CamelCreatorLabelNamespace)
	}
	if lbs[kubernetes.CamelCreatorLabelVersion] != "" {
		delete(lbs, kubernetes.CamelCreatorLabelVersion)
	}

	lbs[kubernetes.CamelClonedLabelKind] = v1.IntegrationKitKind
	lbs[kubernetes.CamelClonedLabelName] = kit.Name
	lbs[kubernetes.CamelClonedLabelNamespace] = kit.Namespace
	lbs[kubernetes.CamelClonedLabelVersion] = kit.ResourceVersion

	return lbs
}

func (o *promoteCmdOptions) editPipe(kb *v1.Pipe, it *v1.Integration, kit *v1.IntegrationKit) (*v1.Pipe, *v1.IntegrationKit) {
	contImage := it.Status.Image
	// IntegrationKit
	dstKit := v1.NewIntegrationKit(o.To, kit.Name)
	dstKit.Spec = *kit.Spec.DeepCopy()
	dstKit.Annotations = cloneAnnotations(kit.Annotations, o.ToOperator)
	dstKit.Labels = cloneLabels(kit.Labels)
	dstKit.Labels = alterKitLabels(dstKit.Labels, kit)
	dstKit.Spec.Image = contImage
	// Pipe
	dst := v1.NewPipe(o.To, kb.Name)
	dst.Spec = *kb.Spec.DeepCopy()
	dst.Annotations = cloneAnnotations(kb.Annotations, o.ToOperator)
	dst.Labels = cloneLabels(kb.Labels)
	if dst.Spec.Integration == nil {
		dst.Spec.Integration = &v1.IntegrationSpec{}
	}
	dst.Spec.Integration.IntegrationKit = &corev1.ObjectReference{
		Namespace: dstKit.Namespace,
		Name:      dstKit.Name,
		Kind:      dstKit.Kind,
	}

	if dst.Spec.Source.Ref != nil {
		dst.Spec.Source.Ref.Namespace = o.To
	}
	if dst.Spec.Sink.Ref != nil {
		dst.Spec.Sink.Ref.Namespace = o.To
	}
	if dst.Spec.Steps != nil {
		for _, step := range dst.Spec.Steps {
			if step.Ref != nil {
				step.Ref.Namespace = o.To
			}
		}
	}

	// We must provide the classpath expected for the IntegrationKit. This is calculated dynamically and
	// would get lost when creating the promoted IntegrationKit (which is in .status.artifacts). For this reason
	// we must report it in the promoted Integration.
	kitClasspath := kit.Status.GetDependenciesPaths()
	if len(kitClasspath) > 0 {
		if dst.Spec.Integration.Traits.JVM == nil {
			dst.Spec.Integration.Traits.JVM = &traitv1.JVMTrait{}
		}
		jvmTrait := dst.Spec.Integration.Traits.JVM
		if jvmTrait.Classpath != "" {
			jvmTrait.Classpath += ":"
		}
		jvmTrait.Classpath += strings.Join(kitClasspath, ":")
	}

	return &dst, dstKit
}

func (o *promoteCmdOptions) replaceResource(res k8sclient.Object) (bool, error) {
	return kubernetes.ReplaceResource(o.Context, o._client, res)
}

func (o *promoteCmdOptions) isDryRun() bool {
	return o.OutputFormat != "" || o.Image
}

// RoleBinding is required to allow access to images in one namespace
// by another namespace. Without this on rbac-enabled clusters, the
// image cannot be pulled.
func addSystemPullerRoleBinding(ctx context.Context, c client.Client, sourceNS string, destNS string) error {
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-image-puller", destNS),
			Namespace: sourceNS,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: destNS,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "system:image-puller",
		},
	}
	applier := c.ServerOrClientSideApplier()
	err := applier.Apply(ctx, rb)

	return err
}

func showImageOnly(cmd *cobra.Command, integration *v1.Integration) {
	fmt.Fprintln(cmd.OutOrStdout(), integration.Status.Image)
}
