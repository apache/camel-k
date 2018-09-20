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

package local

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/apache/camel-k/pkg/util/maven"

	// import openshift utilities
	_ "github.com/apache/camel-k/pkg/util/openshift"
	"github.com/apache/camel-k/version"
)

type localBuilder struct {
	buffer    chan buildOperation
	namespace string
}

type buildOperation struct {
	request build.Request
	output  chan build.Result
}

// NewLocalBuilder create a new builder
func NewLocalBuilder(ctx context.Context, namespace string) build.Builder {
	builder := localBuilder{
		buffer:    make(chan buildOperation, 100),
		namespace: namespace,
	}
	go builder.buildCycle(ctx)
	return &builder
}

func (b *localBuilder) Build(request build.Request) <-chan build.Result {
	res := make(chan build.Result, 1)
	op := buildOperation{
		request: request,
		output:  res,
	}
	b.buffer <- op
	return res
}

func (b *localBuilder) buildCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			b.buffer = nil
			return
		case op := <-b.buffer:
			now := time.Now()
			logrus.Info("Starting new build")
			res := b.execute(&op.request)
			elapsed := time.Now().Sub(now)

			if res.Error != nil {
				logrus.Error("Error during build (total time ", elapsed.Seconds(), " seconds): ", res.Error)
			} else {
				logrus.Info("Build completed in ", elapsed.Seconds(), " seconds")
			}

			op.output <- res
		}
	}
}

func (b *localBuilder) execute(request *build.Request) build.Result {
	project, err := generateProject(request)
	if err != nil {
		return build.Result{
			Error:  err,
			Status: build.StatusError,
		}
	}

	res, err := maven.Build(project)
	if err != nil {
		return build.Result{
			Error:  err,
			Status: build.StatusError,
		}
	}

	logrus.Info("Created tar file ", res.TarFilePath)

	image, err := b.publish(res.TarFilePath, request)
	if err != nil {
		return build.Result{
			Error:  errors.Wrap(err, "could not publish docker image"),
			Status: build.StatusError,
		}
	}

	return build.Result{
		Request:   request,
		Image:     image,
		Error:     nil,
		Status:    build.StatusCompleted,
		Classpath: res.Classpath,
	}
}

func (b *localBuilder) publish(tarFile string, source *build.Request) (string, error) {
	bc := buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: buildv1.SchemeGroupVersion.String(),
			Kind:       "BuildConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + source.Identifier.Name,
			Namespace: b.namespace,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceBinary,
				},
				Strategy: buildv1.BuildStrategy{
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: v1.ObjectReference{
							Kind: "DockerImage",
							Name: "fabric8/s2i-java:2.3",
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &v1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "camel-k-" + source.Identifier.Name + ":" + source.Identifier.Qualifier,
					},
				},
			},
		},
	}

	sdk.Delete(&bc)
	err := sdk.Create(&bc)
	if err != nil {
		return "", errors.Wrap(err, "cannot create build config")
	}

	is := imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "camel-k-" + source.Identifier.Name,
			Namespace: b.namespace,
		},
		Spec: imagev1.ImageStreamSpec{
			LookupPolicy: imagev1.ImageLookupPolicy{
				Local: true,
			},
		},
	}

	sdk.Delete(&is)
	err = sdk.Create(&is)
	if err != nil {
		return "", errors.Wrap(err, "cannot create image stream")
	}

	resource, err := ioutil.ReadFile(tarFile)
	if err != nil {
		return "", errors.Wrap(err, "cannot fully read tar file "+tarFile)
	}

	restClient, err := customclient.GetClientFor("build.openshift.io", "v1")
	if err != nil {
		return "", err
	}

	result := restClient.Post().
		Namespace(b.namespace).
		Body(resource).
		Resource("buildconfigs").
		Name("camel-k-" + source.Identifier.Name).
		SubResource("instantiatebinary").
		Do()

	if result.Error() != nil {
		return "", errors.Wrap(result.Error(), "cannot instantiate binary")
	}

	data, err := result.Raw()
	if err != nil {
		return "", errors.Wrap(err, "no raw data retrieved")
	}

	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(data)
	if err != nil {
		return "", errors.Wrap(err, "cannot unmarshal instantiate binary response")
	}

	ocbuild, err := k8sutil.RuntimeObjectFromUnstructured(&u)
	if err != nil {
		return "", err
	}

	err = kubernetes.WaitCondition(ocbuild, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*buildv1.Build); ok {
			if val.Status.Phase == buildv1.BuildPhaseComplete {
				return true, nil
			} else if val.Status.Phase == buildv1.BuildPhaseCancelled ||
				val.Status.Phase == buildv1.BuildPhaseFailed ||
				val.Status.Phase == buildv1.BuildPhaseError {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, 5*time.Minute)

	err = sdk.Get(&is)
	if err != nil {
		return "", err
	}

	if is.Status.DockerImageRepository == "" {
		return "", errors.New("dockerImageRepository not available in ImageStream")
	}
	return is.Status.DockerImageRepository + ":" + source.Identifier.Qualifier, nil
}

func generateProject(source *build.Request) (maven.Project, error) {
	project := maven.Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
		GroupID:           "org.apache.camel.k.integration",
		ArtifactID:        "camel-k-integration",
		Version:           version.Version,
		DependencyManagement: maven.DependencyManagement{
			Dependencies: maven.Dependencies{
				Dependencies: []maven.Dependency{
					{
						//TODO: camel version should be retrieved from an external request or provided as static version
						GroupID:    "org.apache.camel",
						ArtifactID: "camel-bom",
						Version:    "2.22.1",
						Type:       "pom",
						Scope:      "import",
					},
				},
			},
		},
		Dependencies: maven.Dependencies{
			Dependencies: make([]maven.Dependency, 0),
		},
	}

	//
	// set-up dependencies
	//

	deps := &project.Dependencies
	deps.AddGAV("org.apache.camel.k", "camel-k-runtime-jvm", version.Version)

	for _, d := range source.Dependencies {
		if strings.HasPrefix(d, "camel:") {
			artifactID := strings.TrimPrefix(d, "camel:")

			if !strings.HasPrefix(artifactID, "camel-") {
				artifactID = "camel-" + artifactID
			}

			deps.AddGAV("org.apache.camel", artifactID, "")
		} else if strings.HasPrefix(d, "mvn:") {
			mid := strings.TrimPrefix(d, "mvn:")
			gav := strings.Replace(mid, "/", ":", -1)

			deps.AddEncodedGAV(gav)
		} else {
			return maven.Project{}, fmt.Errorf("unknown dependency type: %s", d)
		}
	}

	return project, nil
}
