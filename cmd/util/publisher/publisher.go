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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/apache/camel-k/deploy"
	"github.com/apache/camel-k/pkg/apis"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
)

// Publishes predefined images for all Camel components
func main() {
	scheme := clientscheme.Scheme
	panicIfErr(apis.AddToScheme(scheme))

	platRun, err := kubernetes.LoadResourceFromYaml(scheme, deploy.Resources["platform-cr.yaml"])
	panicIfErr(err)

	p := platRun.(*v1alpha1.IntegrationPlatform)

	for _, a := range camel.Runtime.Artifacts {
		if a.GroupID == "org.apache.camel" {
			component := strings.TrimPrefix(a.ArtifactID, "camel-")
			build(component, p.Spec.Build.CamelVersion)
		}
	}
}

func build(component string, camelVersion string) {
	dir, err := ioutil.TempDir(os.TempDir(), "camel-k-build-")
	panicIfErr(err)
	defer panicIfErr(os.RemoveAll(dir))

	ctx := builder.Context{
		C:    context.TODO(),
		Path: dir,
		Request: builder.Request{
			Platform: v1alpha1.IntegrationPlatformSpec{
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					CamelVersion: camelVersion,
				},
			},
			Dependencies: []string{
				"camel-k:knative",
				"camel:core",
				"runtime:jvm",
				"runtime:yaml",
				"camel:" + component,
			},
		},
	}

	panicIfErr(builder.GenerateProject(&ctx))
	panicIfErr(builder.ComputeDependencies(&ctx))
	panicIfErr(builder.StandardPackager(&ctx))

	archiveDir, archiveName := filepath.Split(ctx.Archive)
	dockerfile := `
		FROM fabric8/s2i-java:2.3
		ADD ` + archiveName + ` /deployments/
	`
	panicIfErr(ioutil.WriteFile(path.Join(archiveDir, "Dockerfile"), []byte(dockerfile), 0777))
	image := builder.PredefinedImageNameFor(component)
	buildCmd := exec.Command("docker", "build", "-t", image, archiveDir)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	panicIfErr(buildCmd.Run())

	pushCmd := exec.Command("docker", "push", image)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	panicIfErr(pushCmd.Run())
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
