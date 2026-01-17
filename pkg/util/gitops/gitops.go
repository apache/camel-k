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

package util

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/io"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/sets"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const kustomizationContent = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
`

// EditIntegration is in charge to create an Integration with the content required by the GitOps operations.
func EditIntegration(it *v1.Integration, kit *v1.IntegrationKit, toNamespace, toOperator string) *v1.Integration {
	contImage := it.Status.Image
	// Integration
	dstIt := v1.NewIntegration(toNamespace, it.Name)
	dstIt.Spec = *it.Spec.DeepCopy()
	dstIt.Annotations = cloneAnnotations(it.Annotations, toOperator)
	dstIt.Labels = cloneLabels(it.Labels)
	dstIt.Spec.IntegrationKit = nil
	if it.Status.Traits != nil {
		dstIt.Spec.Traits = *it.Status.Traits
	}
	// We make sure not to propagate further the gitops trait
	// to avoid infinite loops. If the user wants to do a chain based
	// strategy, she can use the patch-integration and continue the chain on purpose
	dstIt.Spec.Traits.GitOps = nil
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
		mergedClasspath := getClasspath(kit, jvmTrait)
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
func getClasspath(kit *v1.IntegrationKit, jvmTraitSpec *traitv1.JVMTrait) string {
	jvmTraitClasspath := ""
	if jvmTraitSpec != nil {
		jvmTraitClasspath = jvmTraitSpec.Classpath
	}
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
	for cp := range strings.SplitSeq(cps, ":") {
		s.Add(cp)
	}

	return s
}

// Return all annotations overriding the operator Id if provided.
func cloneAnnotations(ann map[string]string, operatorID string) map[string]string {
	operatorIDAnnotationSet := false
	newMap := make(map[string]string)
	for k, v := range ann {
		if k == "kubectl.kubernetes.io/last-applied-configuration" {
			continue
		}
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
	maps.Copy(newMap, lbs)

	return newMap
}

// EditPipe is in charge to create a Pipe with the content required by the GitOps operations.
func EditPipe(kb *v1.Pipe, it *v1.Integration, kit *v1.IntegrationKit, toNamespace, toOperator string) *v1.Pipe {
	contImage := it.Status.Image
	// Pipe
	dst := v1.NewPipe(toNamespace, kb.Name)
	dst.Spec = *kb.Spec.DeepCopy()
	dst.Annotations = cloneAnnotations(kb.Annotations, toOperator)
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
		if traits.JVM == nil {
			traits.JVM = &traitv1.JVMTrait{}
		}
		jvmTrait := traits.JVM
		mergedClasspath := getClasspath(kit, jvmTrait)
		jvmTrait.Classpath = mergedClasspath
		// We must also set the runtime version so we pin it to the given catalog on which
		// the container image was built
		if traits.Camel == nil {
			traits.Camel = &traitv1.CamelTrait{}
		}
		traits.Camel.RuntimeVersion = kit.Status.RuntimeVersion
	}
	dst.SetTraits(traits)

	if dst.Spec.Source.Ref != nil {
		dst.Spec.Source.Ref.Namespace = toNamespace
	}
	if dst.Spec.Sink.Ref != nil {
		dst.Spec.Sink.Ref.Namespace = toNamespace
	}
	if dst.Spec.Steps != nil {
		for _, step := range dst.Spec.Steps {
			if step.Ref != nil {
				step.Ref.Namespace = toNamespace
			}
		}
	}

	return &dst
}

// AppendKustomizeIntegration creates a Kustomize GitOps based directory structure for the chosen Integration.
func AppendKustomizeIntegration(dstIt *v1.Integration, destinationDir string, overwrite bool) error {
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

	newpath = filepath.Join(destinationDir, appFolderName, "base")
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
	// We only create or override the overlay if it was explicitly requested or it does not exist yet.
	if overwrite || !dirExists(newpath) {
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
	}

	return err
}

func dirExists(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

// getIntegrationPatch will filter those traits/configuration we want to include in the Integration patch.
func getIntegrationPatch(baseIt *v1.Integration) *v1.Integration {
	patchedTraits := patchTraits(baseIt.Spec.Traits)

	patchedIt := v1.NewIntegration("", baseIt.Name)
	patchedIt.Spec = v1.IntegrationSpec{
		Traits: patchedTraits,
	}

	return &patchedIt
}

// getPipePatch will filter those traits/configuration we want to include in the Pipe patch.
func getPipePatch(basePipe *v1.Pipe) *v1.Pipe {
	patchedTraits := patchTraits(*basePipe.Spec.Traits)

	patchedPipe := v1.NewPipe("", basePipe.Name)
	patchedPipe.Spec = v1.PipeSpec{
		Traits: &patchedTraits,
	}

	return &patchedPipe
}

func patchTraits(baseTraits v1.Traits) v1.Traits {
	patchedTraits := v1.Traits{}
	if baseTraits.Affinity != nil {
		patchedTraits.Affinity = baseTraits.Affinity
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
		patchedTraits.Toleration = baseTraits.Toleration
	}

	return patchedTraits
}

// AppendKustomizePipe creates a Kustomize GitOps based directory structure for the chosen Pipe.
func AppendKustomizePipe(dstPipe *v1.Pipe, destinationDir string, overwrite bool) error {
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

	newpath := filepath.Join(destinationDir, appFolderName, "base")
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
	// We only create or override the overlay if it was explicitly requested or it does not exist yet.
	if overwrite || !dirExists(newpath) {
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
	}

	return err
}

// GitToken read the first secret data provided by the Integration Git Secret.
// Returns an empty string if the secret provided is empty.
func GitToken(ctx context.Context, c client.Client, namespace, secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	sec, err := c.CoreV1().Secrets(namespace).Get(ctx, secret, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, v := range sec.Data {
		if v != nil {
			return string(v), nil
		}
	}

	return "", nil
}

// CloneGitProject is in charge to clone the project from a given Git repo configuration.
// If no secretToken is provided, then, it is assumed the project is public.
func CloneGitProject(gitConf v1.GitConfigSpec, dir, secretToken string) (*git.Repository, error) {
	depth := 1
	if gitConf.Commit != "" {
		// only the commit checkout requires full git project history
		depth = 0
	}
	gitCloneOptions := &git.CloneOptions{
		URL:   gitConf.URL,
		Depth: depth,
	}

	if gitConf.Branch != "" {
		if gitConf.Tag != "" {
			return nil, errors.New("illegal arguments: cannot specify both git branch and tag")
		}
		if gitConf.Commit != "" {
			return nil, errors.New("illegal arguments: cannot specify both git branch and commit")
		}
		gitCloneOptions.ReferenceName = plumbing.NewBranchReferenceName(gitConf.Branch)
		gitCloneOptions.SingleBranch = true
	} else if gitConf.Tag != "" {
		if gitConf.Commit != "" {
			return nil, errors.New("illegal arguments: cannot specify both git tag and commit")
		}
		gitCloneOptions.ReferenceName = plumbing.NewTagReferenceName(gitConf.Tag)
		gitCloneOptions.SingleBranch = true
	}

	if secretToken != "" {
		gitCloneOptions.Auth = &http.BasicAuth{
			Username: "camel-k", // yes, this can be anything except an empty string
			Password: secretToken,
		}
	}

	repo, err := git.PlainClone(dir, false, gitCloneOptions)
	if err != nil {
		return nil, err
	}

	if gitConf.Commit != "" {
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, err
		}
		commitHash := plumbing.NewHash(gitConf.Commit)
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: commitHash,
		})
		if err != nil {
			return nil, err
		}
	}

	return repo, nil
}
