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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform/images"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
)

// PublisherOptions --
type PublisherOptions struct {
	StartWith     string
	EndWith       string
	BuildAttempts int
}

// Publishes predefined images for all Camel components
func main() {
	options := PublisherOptions{}

	var cmd = cobra.Command{
		Use:   "publisher",
		Short: "Publisher allows to publish base images before a release",
		Run:   options.run,
	}

	cmd.Flags().StringVar(&options.StartWith, "start-with", "", "The component to start with")
	cmd.Flags().StringVar(&options.EndWith, "end-with", "", "The component to end with")
	cmd.Flags().IntVar(&options.BuildAttempts, "attempts", 5, "The maximum number of build attempts for each image")

	panicIfErr(cmd.Execute())
}

func (options *PublisherOptions) run(cmd *cobra.Command, args []string) {
	scheme := clientscheme.Scheme
	panicIfErr(apis.AddToScheme(scheme))

	platRun, err := kubernetes.LoadResourceFromYaml(scheme, deploy.Resources["platform-cr.yaml"])
	panicIfErr(err)

	p := platRun.(*v1alpha1.IntegrationPlatform)

	started := options.StartWith == ""

	keys := make([]string, 0, len(camel.Runtime.Artifacts))
	for k := range camel.Runtime.Artifacts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		a := camel.Runtime.Artifacts[k]
		if a.GroupID == "org.apache.camel" {
			component := strings.TrimPrefix(a.ArtifactID, "camel-")
			if options.StartWith == component {
				started = true
			}

			if started {
				fmt.Printf("building component %s\n", component)
				options.buildWithAttempts(component, p.Spec.Build.CamelVersion)
			} else {
				fmt.Printf("skipping component %s\n", component)
			}

			if options.EndWith == component {
				fmt.Println("reached final component")
				break
			}
		}
	}
}

func (options *PublisherOptions) buildWithAttempts(component string, camelVersion string) {
	var err error
	for i := 0; i < options.BuildAttempts; i++ {
		err = options.build(component, camelVersion)
		if err != nil {
			sleepTime := 5 * (i + 1)
			fmt.Printf("waiting %d seconds to recover from error %v\n", sleepTime, err)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		} else {
			return
		}
	}
	panicIfErr(errors.Wrap(err, "build failed after maximum number of attempts"))
}

func (options *PublisherOptions) build(component string, camelVersion string) error {
	dir, err := ioutil.TempDir(os.TempDir(), "camel-k-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	dependencies := make([]string, 0)
	for d := range images.StandardDependencies {
		dependencies = append(dependencies, d)
	}
	dependencies = append(dependencies, images.BaseDependency)
	dependencies = append(dependencies, "camel:"+component)

	ctx := builder.Context{
		C:    context.TODO(),
		Path: dir,
		Request: builder.Request{
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: camelVersion,
				},
			},
			Dependencies: dependencies,
		},
	}

	err = builder.GenerateProject(&ctx)
	if err != nil {
		return err
	}
	err = builder.ComputeDependencies(&ctx)
	if err != nil {
		return err
	}
	err = builder.StandardPackager(&ctx)
	if err != nil {
		return err
	}

	archiveDir, archiveName := filepath.Split(ctx.Archive)
	// nolint: gosec
	dockerfile := `
		FROM fabric8/s2i-java:3.0-java8
		ADD ` + archiveName + ` /deployments/
	`

	err = ioutil.WriteFile(path.Join(archiveDir, "Dockerfile"), []byte(dockerfile), 0777)
	if err != nil {
		return err
	}

	image := images.PredefinedImageNameFor(component)
	buildCmd := exec.Command("docker", "build", "-t", image, archiveDir)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	err = buildCmd.Run()
	if err != nil {
		return err
	}

	pushCmd := exec.Command("docker", "push", image)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	err = pushCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
