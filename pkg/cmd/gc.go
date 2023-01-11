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
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apache/camel-k/pkg/platform"
	platformutil "github.com/apache/camel-k/pkg/platform"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/google/go-containerregistry/pkg/name"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"

	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/stream"

	remote "github.com/google/go-containerregistry/pkg/v1/remote"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/digest"
)

type KitNode struct {
	Kit            *v1.IntegrationKit
	Parent         *KitNode
	Children       []*KitNode
	Used           bool
	usedByChildren int
}

func newCmdGC(rootCmdOptions *RootCmdOptions) (*cobra.Command, *gcCmdOptions) {
	options := gcCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "gc",
		Short: "Garbage collect unused resources.",
		Long:  `Delete all unused resources. IntegrationKits that aren't referenced by integrations will be removed.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := decode(&options)(cmd, args); err != nil {
				return err
			}
			return options.preRun(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.run(cmd, args)
		},
	}
	cmd.Flags().BoolP("assumeyes", "y", false, "Do not ask user to confirm resources to be deleted")
	cmd.Flags().BoolP("dry-run", "d", false, "Only list resources to be deleted without removing them")
	cmd.Flags().BoolP("remove-images", "r", false, "When set to true, unused images will be deleted from the image registry. Image layers that are not used will be squashed and new images will created. Integrations whose Image changed will be redeployed. Please make sure you have push and delete rights in the Image Regsitry beforehand.")

	return &cmd, &options
}

type gcCmdOptions struct {
	// TODO: add option to list namespaces when searching for integrations and integrationkits (due to promote feature when changing)
	*RootCmdOptions
	KitsToDelete []*KitNode
	KitsToSquash [][]*KitNode
	UsedImages   map[string][]v1.Integration
	AssumeYes    bool `mapstructure:"assumeyes"`
	DryRun       bool `mapstructure:"dry-run"`
	RemoveImages bool `mapstructure:"remove-images"`
}

func (o *gcCmdOptions) preRun(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	kits, err := o.getKits(c)
	if err != nil {
		return err
	}
	integrations, err := o.getIntegrations(c)
	if err != nil {
		return err
	}
	// let's say that an image is in use if it is referenced by an integration
	o.UsedImages = getUsedImages(integrations)

	o.KitsToDelete = make([]*KitNode, 0)
	o.KitsToSquash = make([][]*KitNode, 0)

	if !o.RemoveImages {
		// simply delete kits not referenced by an integration
		for _, kit := range kits {
			curr := kit
			if o.UsedImages[kit.Status.Image] == nil {
				node := &KitNode{
					Kit:  &curr,
					Used: false,
				}
				o.KitsToDelete = append(o.KitsToDelete, node)
			}
		}
	} else {
		// let's build some trees where nodes are integrationkits. This will help us decide which ones to: keep, delete or squash
		roots, err := buildTrees(kits, o.UsedImages)
		if err != nil {
			return err
		}
		for _, root := range roots {
			toDelete, toSquash := o.trimTree(root)
			o.KitsToDelete = append(o.KitsToDelete, toDelete...)
			o.KitsToSquash = append(o.KitsToSquash, toSquash...)
		}
	}
	o.printInfo(cmd)
	if o.DryRun || o.AssumeYes || o.nothingToDo() {
		return nil
	}
	o.DryRun = ask(cmd)
	return nil
}

func (o *gcCmdOptions) run(cmd *cobra.Command, args []string) error {
	if o.DryRun || o.nothingToDo() {
		return nil
	}
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	// cache is used to check if an Image registry is insecure or not. This
	// avoids us having to retrieve the platform object every time
	cache := make(map[string][]name.Option)
	err = o.squashLayers(o.KitsToSquash, c, cache, o.UsedImages)
	if err != nil {
		return err
	}
	err = o.deleteKits(o.KitsToDelete, c, cache)
	if err != nil {
		return err
	}
	return nil
}

func (o *gcCmdOptions) nothingToDo() bool {
	return len(o.KitsToSquash) == 0 && len(o.KitsToDelete) == 0
}

func (o *gcCmdOptions) printInfo(cmd *cobra.Command) {
	if o.nothingToDo() {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to do")
		return
	}
	if len(o.KitsToSquash) != 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Integration Kits will be squashed:")
	}
	for _, kits := range o.KitsToSquash {
		for _, kit := range kits {
			fmt.Fprint(cmd.OutOrStdout(), fmt.Sprintf("%s in namespace: %s, ", kit.Kit.Name, kit.Kit.Namespace))
		}
		fmt.Fprintln(cmd.OutOrStdout(), fmt.Sprintf("will all be squashed into Integration Kit: %s in namespace: %s", kits[0].Kit.Name, kits[0].Kit.Namespace))
	}
	if len(o.KitsToSquash) != 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Integrations will updated with a new squashed Image and redeployed:")
		for _, kits := range o.KitsToSquash {
			kit := kits[0]
			integrations := o.UsedImages[kit.Kit.Status.Image]
			for _, integration := range integrations {
				fmt.Fprintln(cmd.OutOrStdout(), fmt.Sprintf("%s in namespace: %s", integration.Name, integration.Namespace))
			}
		}
	}
	if len(o.KitsToDelete) != 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Integration Kits will be deleted:")
	}
	for _, kit := range o.KitsToDelete {
		fmt.Fprintln(cmd.OutOrStdout(), fmt.Sprintf("%s in namespace: %s", kit.Kit.Name, kit.Kit.Namespace))
	}
	if len(o.KitsToDelete) != 0 && o.RemoveImages {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Images will be deleted from the Image Registry:")
		for _, kit := range o.KitsToDelete {
			fmt.Fprintln(cmd.OutOrStdout(), kit.Kit.Status.Image)
		}
	}
}

func ask(cmd *cobra.Command) bool {
	fmt.Fprintln(cmd.OutOrStdout(), "\nContinue Y/N ?")
	reader := bufio.NewReader(os.Stdin)
	for {
		s, _ := reader.ReadString('\n')
		s = strings.TrimSuffix(s, "\n")
		s = strings.ToLower(s)
		if len(s) > 1 {
			fmt.Fprintln(os.Stderr, "Please enter Y or N")
			continue
		}
		if strings.Compare(s, "n") == 0 {
			return true
		} else if strings.Compare(s, "y") == 0 {
			return false
		} else {
			continue
		}
	}
}

func (*gcCmdOptions) trimTree(root *KitNode) ([]*KitNode, [][]*KitNode) {
	toDelete := make([]*KitNode, 0)
	toSquash := make([][]*KitNode, 0)
	// let's build a post order list of the tree nodes
	postOrder := make([]*KitNode, 0)
	stack := make([]*KitNode, 0)
	push(&stack, root)
	for curr := pop(&stack); curr != nil; curr = pop(&stack) {
		postOrder = append(postOrder, curr)
		for _, node := range curr.Children {
			push(&stack, node)
		}
	}
	// if a layer is used more than once then keep it
	for i := len(postOrder) - 1; i >= 0; i-- {
		curr := postOrder[i]
		for _, node := range curr.Children {
			if node.usedByChildren > 0 || node.Used {
				curr.usedByChildren += 1
			}
		}
	}
	for i := len(postOrder) - 1; i >= 0; i-- {
		curr := postOrder[i]
		if !isUsed(curr) {
			toDelete = append(toDelete, curr)
		} else if canBeFlattened(curr, curr.Parent) {
			// opportunity to squash with Parents
			squash := make([]*KitNode, 0)
			squash = append(squash, curr)
			for next := curr.Parent; canBeFlattened(curr, next); next = next.Parent {
				squash = append(squash, next)
			}
			toSquash = append(toSquash, squash)
		}
	}
	return toDelete, toSquash
}

// This will squash all unused layers in an image into one layer and replace it in the image registry
func (o *gcCmdOptions) squashLayers(toFlatten [][]*KitNode, c client.Client, cache map[string][]name.Option, usedImages map[string][]v1.Integration) error {
	for _, kitnodes := range toFlatten {
		child := kitnodes[0]
		parent := kitnodes[len(kitnodes)-1]
		// Let's flatten all the layers between the child image and the parent image into a single layer
		// and create a new image from that. That way only relevant changes are kept
		childTag, err := o.getTag(c, child.Kit, cache)
		if err != nil {
			return err
		}
		parentTag, err := o.getTag(c, parent.Kit, cache)
		if err != nil {
			return err
		}
		childImg, err := remote.Image(childTag)
		if err != nil {
			return err
		}
		parentImg, err := remote.Image(parentTag)
		if err != nil {
			return err
		}
		childImgLayers, err := childImg.Layers()
		if err != nil {
			return err
		}
		parentImgLayers, err := parentImg.Layers()
		if err != nil {
			return err
		}
		// optional check, we may remove this in the future
		err = checkParentChildIntegrity(parentImgLayers, childImgLayers, childImg, parentImg)
		if err != nil {
			return err
		}
		childConfig, err := childImg.ConfigFile()
		if err != nil {
			return err
		}
		squashedImage, err := mutate.Config(empty.Image, *childConfig.Config.DeepCopy())
		if err != nil {
			return err
		}
		parentConfig, err := parentImg.ConfigFile()
		if err != nil {
			return err
		}
		squashedImage, err = mutate.Append(squashedImage, createAddendums(parentConfig.History, parentImgLayers)...)
		if err != nil {
			return err
		}
		// copy image annotations
		m, err := childImg.Manifest()
		if err != nil {
			return err
		}
		if len(m.Annotations) != 0 {
			squashedImage = mutate.Annotations(squashedImage, m.Annotations).(gcrv1.Image)
		}
		childDigest, err := childImg.Digest()
		if err != nil {
			return err
		}
		parentDigest, err := parentImg.Digest()
		if err != nil {
			return err
		}
		// squash history
		squashedHistory, err := json.Marshal(childConfig.History[len(parentConfig.History):])
		if err != nil {
			return err
		}
		// squash layers into one single layer
		layer, err := squashLayers(childImgLayers, parentImgLayers)
		if err != nil {
			return err
		}
		squashedImage, err = mutate.Append(squashedImage, mutate.Addendum{
			Layer: layer,
			History: gcrv1.History{
				CreatedBy: fmt.Sprintf("Flattened Image layers %s through %s into a single layer", parentDigest, childDigest),
				Comment:   string(squashedHistory),
			},
		})
		if err != nil {
			return err
		}
		// write the layer and manifest to the Image repository then the image
		err = remote.WriteLayer(childTag.Context(), layer)
		if err != nil {
			return err
		}
		rebasedDigest, err := squashedImage.Digest()
		if err != nil {
			return err
		}
		options := []name.Option{name.StrictValidation}
		if childTag.Context().Registry.Scheme() == "http" {
			options = append(options, name.Insecure)
		}
		squashedImgRefstr := childTag.Context().Name() + "@" + rebasedDigest.String()
		squashedImgRef, err := name.ParseReference(squashedImgRefstr, options...)
		if err != nil {
			return err
		}
		err = remote.Write(squashedImgRef, squashedImage)
		if err != nil {
			return err
		}

		// update kit to point to the new image
		target := child.Kit.DeepCopy()
		target.Status.Image = squashedImgRefstr
		target.Status.BaseImage = parent.Kit.Status.BaseImage
		if target.Spec.Image != "" {
			target.Spec.Image = squashedImgRefstr
			target.Status.ObservedGeneration = child.Kit.Generation
			// let's not trigger a rebuild of the kit so let's update the digest as well
			dgst, err := digest.ComputeForIntegrationKit(target)
			if err != nil {
				return err
			}
			target.Status.Digest = dgst
		}
		err = c.Status().Patch(o.Context, target, client.MergeFrom(child.Kit))
		if err != nil {
			return err
		}
		// update base image kits to point to the new image
		for _, child := range child.Children {
			kit := child.Kit
			target := kit.DeepCopy()
			target.Status.BaseImage = squashedImgRefstr
			err = c.Status().Patch(o.Context, target, client.MergeFrom(kit))
			if err != nil {
				return err
			}
		}
		// update integrations to point to the new image
		// this will trigger a redeployment when running
		for _, integration := range usedImages[child.Kit.Status.Image] {
			target := integration.DeepCopy()
			target.Status.Image = squashedImgRefstr
			err = c.Status().Patch(o.Context, target, client.MergeFrom(&integration))
			if err != nil {
				return err
			}
		}
		// delete old image
		remote.Delete(childTag)
	}
	return nil
}

func (o *gcCmdOptions) deleteKits(toDelete []*KitNode, c client.Client, cache map[string][]name.Option) error {
	for _, kitnode := range toDelete {
		kit := kitnode.Kit
		if o.RemoveImages {
			tag, err := o.getTag(c, kit, cache)
			if err != nil {
				return err
			}
			err = remote.Delete(tag)
			if err != nil {
				return err
			}
		}
		c.Delete(o.Context, kit)
	}
	return nil
}
func (o *gcCmdOptions) getIntegrations(c client.Client) ([]v1.Integration, error) {
	list := v1.NewIntegrationList()
	if err := c.List(o.Context, &list, client.InNamespace(o.Namespace)); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve Integrations from namespace %s", o.Namespace))
	}
	return list.Items, nil
}

func (o *gcCmdOptions) getKits(c client.Client) ([]v1.IntegrationKit, error) {
	list := v1.NewIntegrationKitList()
	if err := c.List(o.Context, &list, client.InNamespace(o.Namespace)); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve IntegrationKits from namespace %s", o.Namespace))
	}
	for _, kit := range list.Items {
		curr := kit
		// let's not run gc when new kits are being built
		if curr.Status.Phase != v1.IntegrationKitPhaseReady && curr.Status.Phase != v1.IntegrationKitPhaseError {
			return nil, errors.New(fmt.Sprintf(`IntegrationKit %s in namespace %s is still building.
			GC command should be run when no new integrations are being created and integrationkits are done building.`, curr.Name, curr.Namespace))
		}
	}
	return list.Items, nil
}

func (o *gcCmdOptions) getTag(c client.Client, kit *v1.IntegrationKit, cache map[string][]name.Option) (name.Reference, error) {
	options, err := o.getPlatformOptions(c, kit, cache)
	if err != nil {
		return nil, err
	}
	return name.ParseReference(kit.Status.Image, options...)
}

func (o *gcCmdOptions) getPlatformOptions(c client.Client, kit *v1.IntegrationKit, cache map[string][]name.Option) ([]name.Option, error) {
	platformName := kit.Status.Platform
	if platformName == "" {
		platformName = platform.DefaultPlatformName
	}
	key := fmt.Sprintf("%s/%s", kit.ObjectMeta.Namespace, platformName)
	options := cache[key]
	if options != nil {
		return options, nil
	}
	platform, err := platformutil.Get(o.Context, c, kit.ObjectMeta.Namespace, platformName)
	if err != nil {
		return nil, err
	}
	options = []name.Option{name.StrictValidation}
	if platform.Status.Build.Registry.Insecure {
		options = append(options, name.Insecure)
	}
	cache[key] = options
	return options, nil
}

func initNodes(kits []v1.IntegrationKit, usedImages map[string][]v1.Integration) (map[string]*KitNode, error) {
	nodes := make(map[string]*KitNode)
	for _, kit := range kits {
		curr := kit
		if curr.Status.Image == "" {
			continue
		}
		node := &KitNode{
			Kit:  &curr,
			Used: usedImages[curr.Status.Image] != nil,
		}
		nodes[curr.Status.Image] = node
	}
	return nodes, nil
}

func buildTrees(kits []v1.IntegrationKit, usedImages map[string][]v1.Integration) ([]*KitNode, error) {
	nodes, err := initNodes(kits, usedImages)
	if err != nil {
		return nil, err
	}
	// let's build parent/child links
	for _, node := range nodes {
		parentImage := node.Kit.Status.BaseImage
		parent := nodes[parentImage]
		// Parent Image is not a Kit
		if parent == nil {
			continue
		}
		// This should not happen, just in case...
		if parent.Kit.Status.Image != parentImage {
			return nil, errors.New(fmt.Sprintf("Base image on kit %s is not referencing base kit's image %s", parent.Kit.Status.Image, parentImage))
		}
		node.Parent = parent
		if parent.Children == nil {
			parent.Children = make([]*KitNode, 0)
		}
		parent.Children = append(parent.Children, node)
	}
	return getRoots(nodes), nil
}

func getRoots(nodes map[string]*KitNode) []*KitNode {
	roots := make([]*KitNode, 0)
	for _, node := range nodes {
		// if a node has no parent then it's the root of a tree
		if node.Parent == nil {
			roots = append(roots, node)
		}
	}
	return roots
}

// A child node can be flattened with it's parent if the child is used but the parent is not
func canBeFlattened(child *KitNode, parent *KitNode) bool {
	return isUsed(child) && parent != nil && !isUsed(parent)
}

// An integrationkit is used if:
// 1) it's image is directly referenced by an integration or
// 2) if it's image is used as a base image for at least two running integration (this means that it contains a layer used at least twice)
func isUsed(node *KitNode) bool {
	return (node.Used || node.usedByChildren > 1)
}

func squashLayers(parent []gcrv1.Layer, child []gcrv1.Layer) (*stream.Layer, error) {
	// create a dummy image with all the extra layers in parent
	newImage, err := mutate.AppendLayers(empty.Image, parent[len(child)-1:]...)
	if err != nil {
		return nil, err
	}
	// now lets squash the layers
	return stream.NewLayer(mutate.Extract(newImage), stream.WithCompressionLevel(gzip.BestCompression)), nil
}

func checkParentChildIntegrity(parentImgLayers []gcrv1.Layer, childImgLayers []gcrv1.Layer, childImg gcrv1.Image, parentImg gcrv1.Image) error {
	if len(parentImgLayers) > len(childImgLayers) {
		return errors.New(fmt.Sprintf("image %q is not based on %q (too few layers)", childImg, parentImg))
	}
	for i, l := range parentImgLayers {
		oldLayerDigest, err := l.Digest()
		if err != nil {
			return err
		}
		origLayerDigest, err := childImgLayers[i].Digest()
		if err != nil {
			return err
		}
		if oldLayerDigest != origLayerDigest {
			return errors.New(fmt.Sprintf("image %q is not based on %q (layer %d mismatch)", childImg, parentImg, i))
		}
	}
	return nil
}

func getUsedImages(integrations []v1.Integration) map[string][]v1.Integration {
	usedImages := make(map[string][]v1.Integration)
	for _, integration := range integrations {
		if usedImages[integration.Status.Image] == nil {
			usedImages[integration.Status.Image] = make([]v1.Integration, 0)
		}
		usedImages[integration.Status.Image] = append(usedImages[integration.Status.Image], integration)
	}
	return usedImages
}

func createAddendums(history []gcrv1.History, layers []gcrv1.Layer) []mutate.Addendum {
	var adds []mutate.Addendum
	layerIndex := 0
	for historyIndex := range history {
		var layer gcrv1.Layer
		emptyLayer := history[historyIndex].EmptyLayer
		if !emptyLayer {
			// empty layers are only in stored history so do not advance layer index
			layer = layers[layerIndex]
			layerIndex++
		}
		adds = append(adds, mutate.Addendum{
			Layer:   layer,
			History: history[historyIndex],
		})
	}
	// In the event history was malformed or non-existent, append the remaining layers.
	for i := layerIndex; i < len(layers); i++ {
		adds = append(adds, mutate.Addendum{Layer: layers[layerIndex]})
	}
	return adds
}

// basic stack impl
func push(nodes *[]*KitNode, node *KitNode) {
	*nodes = append(*nodes, node)
}

func pop(nodes *[]*KitNode) *KitNode {
	if len(*nodes) == 0 {
		return nil
	} else {
		index := len(*nodes) - 1   // Get the index of the top most element.
		element := (*nodes)[index] // Index into the slice and obtain the element.
		*nodes = (*nodes)[:index]  // Remove it from the stack by slicing it off.
		return element
	}
}
