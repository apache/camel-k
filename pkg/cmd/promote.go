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
	netHttp "net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/io"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/sets"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v72/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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
	cmd.Flags().Bool("push-gitops-dir", false, "Commit and push GitOps export directory to git, then create GitHub PR automatically")

	return &cmd, &options
}

type promoteCmdOptions struct {
	*RootCmdOptions
	To            string `mapstructure:"to" yaml:",omitempty"`
	ToOperator    string `mapstructure:"to-operator" yaml:",omitempty"`
	OutputFormat  string `mapstructure:"output" yaml:",omitempty"`
	Image         bool   `mapstructure:"image" yaml:",omitempty"`
	ToGitOpsDir   string `mapstructure:"export-gitops-dir" yaml:",omitempty"`
	PushGitOpsDir bool   `mapstructure:"push-gitops-dir" yaml:",omitempty"`
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
	if o.PushGitOpsDir && o.ToGitOpsDir == "" {
		return errors.New("--push-gitops-dir requires --export-gitops-dir to specify the GitOps directory")
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
		destPipe, err := o.editPipe(sourcePipe, sourceIntegration, sourceKit)
		if err != nil {
			return err
		}
		if o.OutputFormat != "" {
			return showPipeOutput(cmd, destPipe, o.OutputFormat, c.GetScheme())
		}
		if o.ToGitOpsDir != "" {
			err = appendKustomizePipe(destPipe, o.ToGitOpsDir)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), `Exported a Kustomize based Gitops directory to `+o.ToGitOpsDir+` for "`+name+`" Pipe`)
			if o.PushGitOpsDir {
				err = pushGitOpsDirAndOpenPr(destPipe.Name, o.ToGitOpsDir, `"`+name+`" Pipe`, cmd)
				return err
			}
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
	destIntegration := o.editIntegration(sourceIntegration, sourceKit)
	if o.OutputFormat != "" {
		return showIntegrationOutput(cmd, destIntegration, o.OutputFormat)
	}
	if o.ToGitOpsDir != "" {
		err = appendKustomizeIntegration(destIntegration, o.ToGitOpsDir)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), `Exported a Kustomize based Gitops directory to `+o.ToGitOpsDir+` for "`+name+`" Integration`)
		if o.PushGitOpsDir {
			err = pushGitOpsDirAndOpenPr(destIntegration.Name, o.ToGitOpsDir, `"`+name+`" Integration`, cmd)
			return err
		}
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
	if ref == nil {
		return nil, nil
	}
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

func (o *promoteCmdOptions) editIntegration(it *v1.Integration, kit *v1.IntegrationKit) *v1.Integration {
	contImage := it.Status.Image
	// Integration
	dstIt := v1.NewIntegration(o.To, it.Name)
	dstIt.Spec = *it.Spec.DeepCopy()
	dstIt.Annotations = cloneAnnotations(it.Annotations, o.ToOperator)
	dstIt.Labels = cloneLabels(it.Labels)
	dstIt.Spec.IntegrationKit = nil
	if it.Status.Traits != nil {
		dstIt.Spec.Traits = *it.Status.Traits
	}
	if dstIt.Spec.Traits.Container == nil {
		dstIt.Spec.Traits.Container = &traitv1.ContainerTrait{}
	}
	dstIt.Spec.Traits.Container.Image = contImage
	if kit != nil {
		// We must provide the classpath expected for the IntegrationKit. This is calculated dynamically and
		// would get lost when creating the non managed build Integration. For this reason
		// we must report it in the promoted Integration.
		if dstIt.Spec.Traits.JVM == nil {
			dstIt.Spec.Traits.JVM = &traitv1.JVMTrait{}
		}
		jvmTrait := dstIt.Spec.Traits.JVM
		mergedClasspath := getClasspath(kit, jvmTrait.Classpath)
		jvmTrait.Classpath = mergedClasspath
		// We must also set the runtime version so we pin it to the given catalog on which
		// the container image was built
		if dstIt.Spec.Traits.Camel == nil {
			dstIt.Spec.Traits.Camel = &traitv1.CamelTrait{}
		}
		dstIt.Spec.Traits.Camel.RuntimeVersion = kit.Status.RuntimeVersion
	}

	return &dstIt
}

// getClasspath merges the classpath required by the kit with any value provided in the trait.
func getClasspath(kit *v1.IntegrationKit, jvmTraitClasspath string) string {
	kitClasspathSet := kit.Status.GetDependenciesPaths()
	if !kitClasspathSet.IsEmpty() {
		if jvmTraitClasspath != "" {
			jvmTraitClasspathSet := getClasspathSet(jvmTraitClasspath)
			kitClasspathSet = sets.Union(kitClasspathSet, jvmTraitClasspathSet)
		}
		classPaths := kitClasspathSet.List()
		sort.Strings(classPaths)

		return strings.Join(classPaths, ":")
	}

	return jvmTraitClasspath
}

func getClasspathSet(cps string) *sets.Set {
	s := sets.NewSet()
	for _, cp := range strings.Split(cps, ":") {
		s.Add(cp)
	}

	return s
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

func (o *promoteCmdOptions) editPipe(kb *v1.Pipe, it *v1.Integration, kit *v1.IntegrationKit) (*v1.Pipe, error) {
	contImage := it.Status.Image
	// Pipe
	dst := v1.NewPipe(o.To, kb.Name)
	dst.Spec = *kb.Spec.DeepCopy()
	dst.Annotations = cloneAnnotations(kb.Annotations, o.ToOperator)
	dst.Labels = cloneLabels(kb.Labels)
	traits := it.Status.Traits
	if traits == nil {
		traits = &v1.Traits{}
	}
	if traits.Container == nil {
		traits.Container = &traitv1.ContainerTrait{}
	}
	traits.Container.Image = contImage
	if kit != nil {
		// We must provide the classpath expected for the IntegrationKit. This is calculated dynamically and
		// would get lost when creating the non managed build Integration. For this reason
		// we must report it in the promoted Integration.
		mergedClasspath := getClasspath(kit, dst.Annotations[fmt.Sprintf("%sjvm.classpath", v1.TraitAnnotationPrefix)])
		if traits.JVM == nil {
			traits.JVM = &traitv1.JVMTrait{}
		}
		traits.JVM.Classpath = mergedClasspath
		// We must also set the runtime version so we pin it to the given catalog on which
		// the container image was built
		if traits.Camel == nil {
			traits.Camel = &traitv1.CamelTrait{}
		}
		traits.Camel.RuntimeVersion = kit.Status.RuntimeVersion
	}
	if err := dst.SetTraits(traits); err != nil {
		return nil, err
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

	return &dst, nil
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

const kustomizationContent = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`

const baseOverlayDirName = "base"

// appendKustomizeIntegration creates a Kustomize GitOps based directory structure for the chosen Integration.
func appendKustomizeIntegration(dstIt *v1.Integration, destinationDir string) error {
	namespaceDest := dstIt.Namespace
	if _, err := os.Stat(destinationDir); err != nil {
		return err
	}

	baseIt := dstIt.DeepCopy()
	baseIt.Namespace = ""
	if baseIt.Annotations != nil {
		delete(baseIt.Annotations, v1.OperatorIDAnnotation)
	}
	appFolderName := strings.ToLower(baseIt.Name)

	newpath := filepath.Join(destinationDir, appFolderName, "routes")
	err := os.MkdirAll(newpath, io.FilePerm755)
	if err != nil {
		return err
	}
	for _, src := range baseIt.OriginalSourcesOnly() {
		srcName := filepath.Join(newpath, src.Name)
		cnt := []byte(src.Content)
		if err := os.WriteFile(srcName, cnt, io.FilePerm755); err != nil {
			return err
		}
	}

	newpath = filepath.Join(destinationDir, appFolderName, baseOverlayDirName)
	err = os.MkdirAll(newpath, io.FilePerm755)
	if err != nil {
		return err
	}
	marshalledIt, err := kubernetes.ToYAML(baseIt)
	if err != nil {
		return err
	}
	filename := "integration.yaml"
	itName := filepath.Join(newpath, filename)
	if err := os.WriteFile(itName, marshalledIt, io.FilePerm755); err != nil {
		return err
	}
	baseKustCnt := kustomizationContent + `- ` + filename
	kustName := filepath.Join(newpath, "kustomization.yaml")
	if err := os.WriteFile(kustName, []byte(baseKustCnt), io.FilePerm755); err != nil {
		return err
	}

	newpath = filepath.Join(destinationDir, appFolderName, "overlays", namespaceDest)
	err = os.MkdirAll(newpath, io.FilePerm755)
	if err != nil {
		return err
	}
	patchName := "patch-integration.yaml"
	patchedIt := getIntegrationPatch(baseIt)
	marshalledPatchIt, err := kubernetes.ToYAML(patchedIt)
	if err != nil {
		return err
	}
	patchFileName := filepath.Join(newpath, patchName)
	if err := os.WriteFile(patchFileName, marshalledPatchIt, io.FilePerm755); err != nil {
		return err
	}
	nsKustCnt := kustomizationContent + `- ../../base`
	nsKustCnt += `
namespace: ` + namespaceDest + `
patches:
- path: patch-integration.yaml
`
	kustName = filepath.Join(newpath, "kustomization.yaml")
	if err := os.WriteFile(kustName, []byte(nsKustCnt), io.FilePerm755); err != nil {
		return err
	}

	return err
}

// getIntegrationPatch will filter those traits/configuration we want to include in the Integration patch.
func getIntegrationPatch(baseIt *v1.Integration) *v1.Integration {
	patchedTraits := v1.Traits{}
	baseTraits := baseIt.Spec.Traits
	if baseTraits.Affinity != nil {
		patchedTraits.Affinity = baseIt.Spec.Traits.Affinity
	}
	if baseTraits.Camel != nil && baseTraits.Camel.Properties != nil {
		patchedTraits.Camel = &traitv1.CamelTrait{
			Properties: baseTraits.Camel.Properties,
		}
	}
	if baseTraits.Container != nil && (baseTraits.Container.RequestCPU != "" || baseTraits.Container.RequestMemory != "" ||
		baseTraits.Container.LimitCPU != "" || baseTraits.Container.LimitMemory != "") {
		patchedTraits.Container = &traitv1.ContainerTrait{
			RequestCPU:    baseTraits.Container.RequestCPU,
			RequestMemory: baseTraits.Container.RequestMemory,
			LimitCPU:      baseTraits.Container.LimitCPU,
			LimitMemory:   baseTraits.Container.LimitMemory,
		}
	}
	if baseTraits.Environment != nil && baseTraits.Environment.Vars != nil {
		patchedTraits.Environment = &traitv1.EnvironmentTrait{
			Vars: baseTraits.Environment.Vars,
		}
	}
	if baseTraits.JVM != nil && baseTraits.JVM.Options != nil {
		patchedTraits.JVM = &traitv1.JVMTrait{
			Options: baseTraits.JVM.Options,
		}
	}
	if baseTraits.Mount != nil && (baseTraits.Mount.Configs != nil || baseTraits.Mount.Resources != nil ||
		baseTraits.Mount.Volumes != nil || baseTraits.Mount.EmptyDirs != nil) {
		patchedTraits.Mount = &traitv1.MountTrait{
			Configs:   baseTraits.Mount.Configs,
			Resources: baseTraits.Mount.Resources,
			Volumes:   baseTraits.Mount.Volumes,
			EmptyDirs: baseTraits.Mount.EmptyDirs,
		}
	}
	if baseTraits.Toleration != nil {
		patchedTraits.Toleration = baseIt.Spec.Traits.Toleration
	}

	patchedIt := v1.NewIntegration("", baseIt.Name)
	patchedIt.Spec = v1.IntegrationSpec{
		Traits: patchedTraits,
	}

	return &patchedIt
}

// appendKustomizePipe creates a Kustomize GitOps based directory structure for the chosen Pipe.
func appendKustomizePipe(dstPipe *v1.Pipe, destinationDir string) error {
	namespaceDest := dstPipe.Namespace
	if _, err := os.Stat(destinationDir); err != nil {
		return err
	}

	basePipe := dstPipe.DeepCopy()
	basePipe.Namespace = ""
	if basePipe.Annotations != nil {
		delete(basePipe.Annotations, v1.OperatorIDAnnotation)
	}
	appFolderName := strings.ToLower(basePipe.Name)

	newpath := filepath.Join(destinationDir, appFolderName, baseOverlayDirName)
	err := os.MkdirAll(newpath, io.FilePerm755)
	if err != nil {
		return err
	}
	marshalledPipe, err := kubernetes.ToYAML(basePipe)
	if err != nil {
		return err
	}
	filename := "pipe.yaml"
	itName := filepath.Join(newpath, filename)
	if err := os.WriteFile(itName, marshalledPipe, io.FilePerm755); err != nil {
		return err
	}
	baseKustCnt := kustomizationContent + `- ` + filename
	kustName := filepath.Join(newpath, "kustomization.yaml")
	if err := os.WriteFile(kustName, []byte(baseKustCnt), io.FilePerm755); err != nil {
		return err
	}

	newpath = filepath.Join(destinationDir, appFolderName, "overlays", namespaceDest)
	err = os.MkdirAll(newpath, io.FilePerm755)
	if err != nil {
		return err
	}
	patchName := "patch-pipe.yaml"
	patchedPipe := getPipePatch(basePipe)
	marshalledPatchPipe, err := kubernetes.ToYAML(patchedPipe)
	if err != nil {
		return err
	}
	patchFileName := filepath.Join(newpath, patchName)
	if err := os.WriteFile(patchFileName, marshalledPatchPipe, io.FilePerm755); err != nil {
		return err
	}
	nsKustCnt := kustomizationContent + `- ../../base`
	nsKustCnt += `
namespace: ` + namespaceDest + `
patches:
- path: patch-pipe.yaml
`
	kustName = filepath.Join(newpath, "kustomization.yaml")
	if err := os.WriteFile(kustName, []byte(nsKustCnt), io.FilePerm755); err != nil {
		return err
	}

	return err
}

// getPipePatch will filter those traits/configuration we want to include in the Pipe patch.
func getPipePatch(basePipe *v1.Pipe) *v1.Pipe {
	patchedPipe := v1.NewPipe("", basePipe.Name)
	patchedPipe.Annotations = basePipe.Annotations
	// Only keep those traits we want to include in the patch
	for kAnn := range basePipe.Annotations {
		if strings.HasPrefix(kAnn, v1.TraitAnnotationPrefix) {
			if !isPipeTraitPatch(kAnn) {
				delete(basePipe.Annotations, kAnn)
			}
		}
	}
	return &patchedPipe
}

// isPipeTraitPatch returns true if it belongs to the list of the opinionated traits we want to keep in the patch.
func isPipeTraitPatch(keyAnnotation string) bool {
	if strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"affinity") {
		return true
	}
	if keyAnnotation == v1.TraitAnnotationPrefix+"camel.properties" {
		return true
	}
	if strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"container.request") ||
		strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"container.limit") {
		return true
	}
	if keyAnnotation == v1.TraitAnnotationPrefix+"environment.vars" {
		return true
	}
	if keyAnnotation == v1.TraitAnnotationPrefix+"jvm.options" {
		return true
	}
	if strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"mount.configs") ||
		strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"mount.resources") ||
		strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"mount.volumes") ||
		strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"mount.empty-dirs") {
		return true
	}
	if strings.HasPrefix(keyAnnotation, v1.TraitAnnotationPrefix+"toleration") {
		return true
	}

	return false
}

func pushGitOpsDirAndOpenPr(promotionName, destinationDir, printName string, cmd *cobra.Command) error {
	appFolderName := strings.ToLower(promotionName)
	basePath := filepath.Join(destinationDir, appFolderName, baseOverlayDirName)
	if _, err := os.Stat(basePath); err != nil {
		return err
	}

	repo, worktree, err := getGitRepoAndWorktree(basePath)
	if err != nil {
		return err
	}

	// Stage changed files in the base overlay directory and detect whether the files are updated or created
	detectedFileChanges, containsNewFiles, err := stageChangedFiles(worktree, appFolderName)
	if err != nil {
		return err
	}
	if !detectedFileChanges {
		_, err = fmt.Fprintln(cmd.OutOrStdout(), `GitOps export for "`+promotionName+`" is already up to date`)
		return err
	}

	// Validate current HEAD reference is a branch
	if err = validateHeadReference(repo, destinationDir); err != nil {
		return err
	}

	// Create and checkout a new git branch used for the GitOps export
	var prBranch string   // the branch where we push changes
	var baseBranch string // target branch
	// reference won't be found for example if there is no initial commit and the git repo is on the default branch
	headRef, err := repo.Head()
	headRefNotAvailable := err != nil
	if !headRefNotAvailable {
		baseBranch = headRef.Name().Short()
		prBranch, err = createBranchForGitOpsExportPush(worktree)
		if err != nil {
			return err
		}
	}

	if err = commitGitChanges(containsNewFiles, worktree, cmd, printName); err != nil {
		return err
	}

	if headRefNotAvailable {
		// we had to defer the creating of a new branch until there was initial commit
		prBranch, err = createBranchForGitOpsExportPush(worktree)
		if err != nil {
			return err
		}
		if headRef, err = repo.Head(); err != nil {
			return err
		} else {
			baseBranch = headRef.Name().Short()
		}
	}

	remote, err := findTargetRepoRemote(repo, cmd)
	if err != nil || remote == nil {
		return err
	}

	if err = pushGitCommit(remote, repo, prBranch); err != nil {
		return err
	}

	return createGitHubPr(remote, cmd, containsNewFiles, printName, baseBranch, prBranch)
}

func createBranchForGitOpsExportPush(worktree *git.Worktree) (string, error) {
	newBranch := fmt.Sprintf("camel-k-gitops-export-%d", time.Now().UnixNano())
	err := worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(newBranch),
		Create: true,
		Keep:   true,
	})
	return newBranch, err
}

func commitGitChanges(containsNewFiles bool, worktree *git.Worktree, cmd *cobra.Command, printName string) error {
	var commitMsg string
	if containsNewFiles {
		commitMsg = fmt.Sprintf("chore: add GitOps base overlay for %s\n\nGenerated by Camel K promote command", printName)
	} else {
		commitMsg = fmt.Sprintf("chore: update GitOps base overlay for %s\n\nGenerated by Camel K promote command", printName)
	}
	_, err := worktree.Commit(commitMsg, &git.CommitOptions{
		AllowEmptyCommits: false,
		Author: &object.Signature{
			Name: "Camel K",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit git changes: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Created git commit with the GitOps base overlay")
	return err
}

func findTargetRepoRemote(repo *git.Repository, cmd *cobra.Command) (*git.Remote, error) {
	var remote *git.Remote
	remotes, err := repo.Remotes()
	if err != nil {
		err = fmt.Errorf("failed to get git remotes: %w", err)
	} else {
		for _, thatRemote := range remotes {
			// prefer the "origin" remote, but fallback to any remote
			if remote == nil || thatRemote.Config().Name == "origin" {
				remote = thatRemote
			}
		}
		if remote == nil {
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Cannot push changes to git: no git remote configured")
		}
	}
	return remote, err
}

func validateHeadReference(repo *git.Repository, destinationDir string) error {
	headRef, err := repo.Storer.Reference(plumbing.HEAD)
	if err != nil {
		return fmt.Errorf("cannot determine current git branch: %w", err)
	}
	if !headRef.Target().IsBranch() {
		return fmt.Errorf(`git repository "%s" HEAD must be a branch, but is: %s`, destinationDir, headRef.Target())
	}
	return nil
}

func stageChangedFiles(worktree *git.Worktree, appFolderName string) (bool, bool, error) {
	var containsNewFiles, detectedFileChanges bool
	status, err := worktree.Status()
	if err != nil {
		err = fmt.Errorf("failed to get git status: %w", err)
	} else {
		baseSubPath := filepath.Join(appFolderName, baseOverlayDirName)
		for file, fileStatus := range status {
			if strings.Contains(file, baseSubPath) && (fileStatus.Worktree == git.Untracked || fileStatus.Worktree == git.Modified) {
				if _, err = worktree.Add(file); err != nil {
					err = fmt.Errorf(`failed to add file "%s" to git repository: %w`, file, err)
					return false, false, err
				}
				if fileStatus.Worktree == git.Untracked {
					containsNewFiles = true
				}
				detectedFileChanges = true
			}
		}
	}
	return detectedFileChanges, containsNewFiles, err
}

func getGitRepoAndWorktree(basePath string) (*git.Repository, *git.Worktree, error) {
	var worktree *git.Worktree
	repo, err := git.PlainOpenWithOptions(basePath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		err = errors.New(`failed to open git repository at path "` + basePath + `": ` + err.Error())
	} else {
		worktree, err = repo.Worktree()
		if err != nil {
			err = fmt.Errorf("failed to get git worktree: %w", err)
		}
	}
	return repo, worktree, err
}

func createGitHubPr(remote *git.Remote, cmd *cobra.Command, containsNewFiles bool, printName string, baseBranch string,
	prBranch string) error {
	owner, repoName, err := parseGitHubURL(remote)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "Warning: Could not create GitHub PR: %v\n", err)
		return nil
	}
	gitHubClient, err := newGitHubClient(cmd.Context())
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "Warning: Could not create GitHub PR: %v\n", err)
		return nil
	}

	var prTitle string
	var prBody string
	if containsNewFiles {
		prTitle = fmt.Sprintf("chore: add GitOps base overlay for %s", printName)
		prBody = fmt.Sprintf("Adds GitOps base overlay for %s. Generated by Camel K promote command.", printName)
	} else {
		prTitle = fmt.Sprintf("chore: update GitOps base overlay for %s", printName)
		prBody = fmt.Sprintf("Updates GitOps base overlay for %s. Generated by Camel K promote command.", printName)
	}
	newPR := &github.NewPullRequest{
		Title:               github.Ptr(prTitle),
		Head:                github.Ptr(prBranch),
		Base:                github.Ptr(baseBranch),
		Body:                github.Ptr(prBody),
		MaintainerCanModify: github.Ptr(true),
	}

	pr, _, err := gitHubClient.PullRequests.Create(cmd.Context(), owner, repoName, newPR)
	if err != nil {
		return fmt.Errorf("cannot create a new GitHub PR: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created GitHub PR with the base overlay from exported GitOps directory: %s\n", pr.GetHTMLURL())
	return err
}

func parseGitHubURL(remote *git.Remote) (string, string, error) {
	repoURL := remote.Config().URLs[0]
	if !strings.Contains(repoURL, "github.com") {
		return "", "", fmt.Errorf("automatic PR creation requires a GitHub repository, but found: %s", repoURL)
	}
	if hasHTTPSGitURLFormat(repoURL) {
		parsedURL, err := url.Parse(repoURL)
		if err != nil {
			return "", "", fmt.Errorf("failed to parse HTTPS URL: %w", err)
		}
		parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub HTTPS URL format: %s", repoURL)
		}
		return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
	}
	if hasSSHGitURLFormat(repoURL) {
		pathParts := strings.SplitN(repoURL, ":", 2)
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL format: %s", repoURL)
		}

		cleanPath := strings.TrimSuffix(path.Clean(pathParts[1]), ".git")
		parts := strings.Split(cleanPath, "/")
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub SSH URL path: %s", cleanPath)
		}
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("unsupported remote URL format: %s", repoURL)
}

func newGitHubClient(ctx context.Context) (*github.Client, error) {
	if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
		httpClient := &netHttp.Client{}
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		ctx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
		httpClient = oauth2.NewClient(ctx, ts)
		return github.NewClient(httpClient), nil
	}
	return nil, errors.New(`cannot create GitHub PR, because the "GITHUB_TOKEN" environment variable is not set`)
}

func hasSSHGitURLFormat(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git@")
}

func hasHTTPSGitURLFormat(repoURL string) bool {
	return strings.HasPrefix(repoURL, "https://")
}

func pushGitCommit(remote *git.Remote, repo *git.Repository, prBranch string) error {
	var auth transport.AuthMethod
	var err error
	if hasSSHGitURLFormat(remote.Config().URLs[0]) {
		if keyPath := os.Getenv("KAMEL_SSH_KEY_PATH"); keyPath != "" {
			auth, err = ssh.NewPublicKeysFromFile("git", keyPath, os.Getenv("KAMEL_SSH_KEY_PASSPHRASE"))
			if err != nil {
				return fmt.Errorf(`failed to load SSH key from "%s": %w`, keyPath, err)
			}
		} else {
			// if there are multiple SSH keys, there is no guarantee which key is used
			auth, err = ssh.NewSSHAgentAuth("git")
			if err != nil {
				return fmt.Errorf("failed to create git SSH agent: %w", err)
			}
		}
	} else if hasHTTPSGitURLFormat(remote.Config().URLs[0]) {
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			auth = &http.BasicAuth{
				Username: "camel-k", // this can be anything except an empty string
				Password: token,
			}
		} else {
			return fmt.Errorf("HTTPS git push requires GITHUB_TOKEN environment variable")
		}
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: remote.Config().Name,
		Auth:       auth,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+%s:%s", plumbing.NewBranchReferenceName(prBranch), plumbing.NewBranchReferenceName(prBranch))),
		},
	})
	if err != nil {
		err = fmt.Errorf(`failed to push branch "%s" to remote "%s": %w`, prBranch, remote.Config().Name, err)
	}
	return err
}
