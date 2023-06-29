//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package cli

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	platformutil "github.com/apache/camel-k/v2/pkg/platform"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/digest"
	"github.com/apache/camel-k/v2/pkg/util/openshift"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/uuid"

	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/e2e/support/util"
)

func TestSquashIntegrationKits(t *testing.T) {
	RegisterTestingT(t)

	ocp, err := openshift.IsOpenShift(TestClient())
	assert.Nil(t, err)
	if ocp {
		t.Skip("Avoid running on OpenShift until Image Registry supports delete operation")
		return
	}
	t.Run("Squash IntegrationKits", func(t *testing.T) {
		// make sure kits are deleted
		Expect(DeleteKits(ns)).To(Succeed())
		tests := []struct {
			title    string
			tree     string
			toDelete string
			toSquash string
		}{
			// Syntax for n ary trees in preorder traversal is NAME(f|t)| where:
			// Name is the name of the kit,
			// t : kit is used
			// f : kit is unused
			// | marks the end of children
			{
				title:    "Nothing to do",
				tree:     "",
				toDelete: "",
				toSquash: "",
			},
			{
				title:    "Single used kit ",
				tree:     "a(t)",
				toDelete: "",
				toSquash: "",
			},
			{
				title:    "Single unused kit",
				tree:     "a(f)",
				toDelete: "a",
				toSquash: "",
			},
			{
				title:    "Basic tree",
				tree:     "a(f)b(t)",
				toDelete: "a",
				toSquash: "ba",
			},
			{
				title:    "Simple tree line",
				tree:     "a(f)b(f)c(f)d(f)e(t)",
				toDelete: "abcd",
				toSquash: "edcba",
			},
			{
				title:    "Tree",
				tree:     "a(f)b(f)e(f)|f(f)k(t)|||c(t)|d(f)g(t)|h(f)|i(f)|j(t)|||",
				toDelete: "befhi",
				toSquash: "kfb",
			},
		}

		for _, curr := range tests {
			thetest := curr
			t.Run(thetest.title, func(t *testing.T) {
				// build kit tree
				nodes := buildKits(thetest.tree, operatorID, ns, t)

				// check dry run
				checkDryRunLogs(ns, thetest.toDelete, thetest.toSquash, nodes, t)

				// check real run
				Expect(Kamel("kit", "squash", "-y", "-n", ns).Execute()).To(Succeed())

				for _, r := range thetest.toDelete {
					kitName := string(r)
					Eventually(Kit(ns, kitName), TestTimeoutShort).Should(BeNil())
					kit := nodes[kitName].Kit
					options, err := getPlatformOptions(TestContext, TestClient(), kit)
					assert.NoError(t, err)
					tag, err := getTag(kit, options)
					assert.NoError(t, err)
					// assert the image was deleted
					_, err = remote.Image(tag)
					assert.True(t, isNotFound(err))
				}
				for _, node := range nodes {
					// It we shouldn't delete it then it should still exist
					if !strings.Contains(thetest.toDelete, node.Name) {
						oldKit := node.Kit
						newKit := Kit(ns, oldKit.Name)()
						assert.NotNil(t, newKit)
						// Image has been squashed then the Image has changed
						if strings.Contains(curr.toSquash, node.Name) {
							assert.True(t, oldKit.Status.Image != newKit.Status.Image)
						}
					}
					if node.Used {
						// check that the integration still runs fine
						for _, message := range node.ExpectedMessages {
							Eventually(IntegrationLogs(ns, node.Name), TestTimeoutShort).Should(ContainSubstring(message))
						}
					}
				}
				// Clean up
				Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
				Expect(DeleteKits(ns)).To(Succeed())

			})
		}
	})
}

func checkDryRunLogs(ns string, toDelete string, toSquash string, nodes map[string]*TestNode, t *testing.T) {
	ctx, cancel := context.WithCancel(TestContext)
	defer cancel()
	piper, pipew := io.Pipe()
	defer pipew.Close()
	defer piper.Close()
	kamelBuild := KamelWithContext(TestContext, "kit", "squash", "-d", "-n", ns)
	kamelBuild.SetOut(pipew)
	kamelBuild.SetErr(pipew)

	redeployed := make([]string, 0)
	squashness := make([]string, 0)
	if len(toSquash) > 0 {
		redeployed = append(redeployed, "The following Integrations will updated with a new squashed Image and redeployed:")
		squashness = append(squashness, "The following Integration Kits will be squashed:")
		for _, s := range strings.Split(toSquash, "|") {
			for _, r := range s {
				squashness = append(squashness, fmt.Sprintf("%s in namespace: %s, ", nodes[string(r)].Kit.Name, ns))
			}
			kit := nodes[string(s[0])].Kit.Name
			squashness = append(squashness, fmt.Sprintf("will all be squashed into Integration Kit:  %s  in namespace:  %s", kit, ns))
			redeployed = append(redeployed, fmt.Sprintf("%s in namespace: %s", kit, ns))
		}
	}
	deleteness := make([]string, 0)
	if len(toDelete) > 0 {
		deleteness = append(deleteness, "The following Integration Kits will be deleted:")
		deleteness = append(deleteness, "The following Images will be deleted from the Image Registry:")
		for _, r := range toDelete {
			node := nodes[string(r)]
			deleteness = append(deleteness, fmt.Sprintf("%s in namespace: %s", node.Kit.Name, ns))
			deleteness = append(deleteness, node.Kit.Status.Image)
		}
	}
	if len(toDelete) == 0 && len(toSquash) == 0 && len(redeployed) == 0 {
		deleteness = append(deleteness, "Nothing to do")
	}
	logScanner := util.NewStrictLogScanner(ctx, piper, true, append(redeployed, append(squashness, deleteness...)...)...)
	go func() {
		err := kamelBuild.Execute()
		assert.NoError(t, err)
		logScanner.Done()
		cancel()
	}()
	Eventually(logScanner.ExactMatch(), TestTimeoutShort).Should(BeTrue())
}

func buildKits(tree string, operatorID string, ns string, t *testing.T) map[string]*TestNode {
	scanner := bufio.NewScanner(strings.NewReader(tree))
	scanner.Split(ScanTree)
	nodes := make(map[string]*TestNode, 0)
	buildKitsRecur(nil, scanner, nodes, operatorID, ns, t)
	return nodes
}

func buildKitsRecur(node *TestNode, scanner *bufio.Scanner, nodes map[string]*TestNode, operatorID string, ns string, t *testing.T) {
	if !scanner.Scan() {
		return
	}
	name := scanner.Text()
	if node == nil {
		if !scanner.Scan() {
			return
		}
		used := scanner.Text()
		node := buildKit(nil, name, used, operatorID, ns, t)

		nodes[name] = node
		buildKitsRecur(node, scanner, nodes, operatorID, ns, t)
		return
	}
	if name == "|" {
		return
	}
	if !scanner.Scan() {
		return
	}
	used := scanner.Text()
	child := buildKit(node, name, used, operatorID, ns, t)
	nodes[child.Name] = child
	node.Children = append(node.Children, child)
	buildKitsRecur(child, scanner, nodes, operatorID, ns, t)
	buildKitsRecur(node, scanner, nodes, operatorID, ns, t)
}

func buildKit(parent *TestNode, nodeName string, used string, operatorID string, ns string, t *testing.T) *TestNode {
	var isUsed = false
	if used == "t" {
		isUsed = true
	}
	message := uuid.NewString()
	node := &TestNode{
		Parent:           parent,
		Name:             nodeName,
		Used:             isUsed,
		Message:          message,
		ExpectedMessages: []string{message},
		Children:         make([]*TestNode, 0),
	}
	if parent == nil {
		node.Message = ""
		node.ExpectedMessages = make([]string, 0)
		// Create a root image. We will then manually create some layers on top of it to simulate a graph of integration kits
		Expect(KamelRunWithID(operatorID, ns, "files/FileRoute.java", "--name", node.Name).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, node.Name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, node.Name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		kit := Kit(ns, IntegrationKit(ns, node.Name)())()
		node.Kit = kit
		if !isUsed {
			Expect(KamelWithContext(TestContext, "delete", node.Name, "-n", ns).Execute()).To(Succeed())
		}
		return node
	}

	node.ExpectedMessages = append(node.ExpectedMessages, node.Parent.ExpectedMessages...)
	node.Kit = createKit(node, t)
	if !isUsed {
		// it if not used then don't create a running integration
		return node
	}
	Expect(KamelRunWithID(operatorID, ns, "files/FileRoute.java", "--name", node.Name, "--kit", node.Name).Execute()).To(Succeed())
	Eventually(IntegrationPodPhase(ns, node.Name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, node.Name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationKit(ns, node.Name), TestTimeoutMedium).Should(Equal(node.Name))
	for _, message := range node.ExpectedMessages {
		Eventually(IntegrationLogs(ns, node.Name), TestTimeoutShort).Should(ContainSubstring(message))
	}
	node.Integration = Integration(ns, node.Name)()
	assert.NotNil(t, (node.Integration))
	return node
}

func getTag(kit *v1.IntegrationKit, options []name.Option) (name.Reference, error) {
	return name.ParseReference(kit.Status.Image, options...)
}

// Creates a new IntegrationKit based base on it's parent
func createKit(node *TestNode, t *testing.T) *v1.IntegrationKit {
	newImgRefstr := createImage(node, t)
	parentKit := node.Parent.Kit
	// Create a kit based on the new image
	kit := v1.NewIntegrationKit(parentKit.Namespace, node.Name)
	// Copy the Spec and Labels from the Parent. This helps camel-k use the kit for future integrations
	kit.Spec = *parentKit.Spec.DeepCopy()
	kit.Spec.Image = newImgRefstr
	kit.Labels = make(map[string]string)
	kit.Labels[v1.IntegrationKitPriorityLabel] = "2000"
	kit.Labels["camel.apache.org/runtime.version"] = parentKit.Labels["camel.apache.org/runtime.version"]
	kit.Labels["camel.apache.org/runtime.provider"] = parentKit.Labels["camel.apache.org/runtime.provider"]
	kit.Labels["camel.apache.org/kit.type"] = "external"
	Expect(TestClient().Create(TestContext, kit)).To(Succeed())
	Eventually(KitPhase(parentKit.Namespace, node.Name), TestTimeoutMedium).Should(Equal(v1.IntegrationKitPhaseReady))

	// Update the base image of the kit so GC builds the correct Kit graph
	kit = Kit(parentKit.Namespace, node.Name)()
	kit.Status.Image = newImgRefstr
	kit.Status.BaseImage = parentKit.Status.Image
	kit.Status.Platform = parentKit.Status.Platform
	dgst, err := digest.ComputeForIntegrationKit(kit)
	assert.NoError(t, err)
	kit.Status.Digest = dgst
	Expect(TestClient().Status().Update(TestContext, kit)).To(Succeed())
	return kit
}

// Creates an Image from the parent Kit by adding a single Layer on top of it
func createImage(node *TestNode, t *testing.T) string {
	parentKit := node.Parent.Kit

	l := createlayer(node, t)
	// write the layer first (plus that way the stream is consumed and we can get the digest)
	options, err := getPlatformOptions(TestContext, TestClient(), parentKit)
	assert.NoError(t, err)
	parentTag, err := getTag(parentKit, options)
	assert.NoError(t, err)
	err = remote.WriteLayer(parentTag.Context(), l)
	assert.NoError(t, err)
	parentImage, err := remote.Image(parentTag)
	assert.NoError(t, err)
	img, err := mutate.AppendLayers(
		parentImage,
		l,
	)
	newImgDigest, err := img.Digest()
	assert.NoError(t, err)
	newImgRefstr := parentTag.Context().Name() + "@" + newImgDigest.String()
	newImgRef, err := name.ParseReference(newImgRefstr, options...)
	assert.NoError(t, err)

	err = remote.Write(newImgRef, img)
	assert.NoError(t, err)
	return newImgRefstr
}

// Creates an Image layer with a single file based on the node name
func createlayer(node *TestNode, t *testing.T) *stream.Layer {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)
	go func() {
		pw.CloseWithError(func() error {
			body := node.Message
			if err := tw.WriteHeader(&tar.Header{
				Name:     fmt.Sprintf("/var/camel/%s.txt", node.Name),
				Mode:     0600,
				Size:     int64(len(body)),
				Typeflag: tar.TypeReg,
			}); err != nil {
				return err
			}
			if _, err := tw.Write([]byte(body)); err != nil {
				return err
			}
			return tw.Close()
		}())
	}()
	return stream.NewLayer(pr, stream.WithCompressionLevel(gzip.BestCompression))
}

func getPlatformOptions(ctx context.Context, c client.Client, kit *v1.IntegrationKit) ([]name.Option, error) {
	platform, err := platformutil.GetOrFindLocal(ctx, c, kit.ObjectMeta.Namespace)
	if err != nil {
		return nil, err
	}
	options := []name.Option{name.StrictValidation}
	if platform.Status.Build.Registry.Insecure {
		options = append(options, name.Insecure)
	}
	return options, nil
}

func ScanTree(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Return nothing if at end of file and no data passed
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	r, width := utf8.DecodeRune(data)
	if r == '|' {
		return width, data[0:width], nil
	}

	if r == '(' {
		// read if it is used or not
		_, width2 := utf8.DecodeRune(data[width:])
		// skpi ahead of next ')'
		_, width3 := utf8.DecodeRune(data[width2:])
		return width + width2 + width3, data[width : width+width2], nil
	}

	// Read until '('
	for pos, width := 0, 0; pos < len(data); pos += width {
		var r rune
		r, width = utf8.DecodeRune(data[pos:])
		if r == '(' {
			return pos, data[0:pos], nil
		}
	}
	return len(data), data, nil
}

func isNotFound(err error) bool {
	if status, ok := err.(*transport.Error); ok || errors.As(err, &status) {
		return status.StatusCode == http.StatusNotFound
	}
	return false
}

type ErrorStatusCode struct {
	error
	StatusCode int
}

type TestNode struct {
	Kit              *v1.IntegrationKit
	Integration      *v1.Integration
	Parent           *TestNode
	Children         []*TestNode
	Used             bool
	Name             string
	Message          string
	ExpectedMessages []string
}
