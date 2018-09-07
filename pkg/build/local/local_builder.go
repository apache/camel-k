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
	"github.com/sirupsen/logrus"
	"time"
	"io/ioutil"
	"github.com/pkg/errors"
	buildv1 "github.com/openshift/api/build/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/core/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/client-go/rest"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"k8s.io/apimachinery/pkg/runtime/schema"
	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	_ "github.com/apache/camel-k/pkg/util/openshift"
	build "github.com/apache/camel-k/pkg/build/api"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/version"
	"github.com/apache/camel-k/pkg/util/maven"
)

type localBuilder struct {
	buffer    chan buildOperation
	namespace string
}

type buildOperation struct {
	source build.BuildSource
	output chan build.BuildResult
}

func NewLocalBuilder(ctx context.Context, namespace string) build.Builder {
	builder := localBuilder{
		buffer:    make(chan buildOperation, 100),
		namespace: namespace,
	}
	go builder.buildCycle(ctx)
	return &builder
}

func (b *localBuilder) Build(source build.BuildSource) <-chan build.BuildResult {
	res := make(chan build.BuildResult, 1)
	op := buildOperation{
		source: source,
		output: res,
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
			image, err := b.execute(op.source)
			elapsed := time.Now().Sub(now)
			if err != nil {
				logrus.Error("Error during build (total time ", elapsed.Seconds(), " seconds): ", err)
			} else {
				logrus.Info("Build completed in ", elapsed.Seconds(), " seconds")
			}

			if err != nil {
				op.output <- build.BuildResult{
					Source: &op.source,
					Status: build.BuildStatusError,
					Error:  err,
				}
			} else {
				op.output <- build.BuildResult{
					Source: &op.source,
					Status: build.BuildStatusCompleted,
					Image:  image,
				}
			}

		}
	}
}

func (b *localBuilder) execute(source build.BuildSource) (string, error) {

	project := maven.Project{
		GroupId:    "org.apache.camel.k.integration",
		ArtifactId: "camel-k-integration",
		Version:    "1.0.0",
		JavaSources: map[string]string{
			"kamel/Routes.java": source.Code,
		},
		Env: map[string]string{
			"JAVA_MAIN_CLASS": "org.apache.camel.k.jvm.Application",
			"KAMEL_CLASS":     "kamel.Routes",
		},
		Dependencies: []maven.Dependency{
			{
				GroupId:    "org.apache.camel.k",
				ArtifactId: "camel-k-runtime-jvm",
				Version:    version.Version,
			},
		},
	}

	tarFileName, err := maven.Build(project)
	if err != nil {
		return "", err
	}
	logrus.Info("Created tar file ", tarFileName)

	image, err := b.publish(tarFileName, source)
	if err != nil {
		return "", errors.Wrap(err, "could not publish docker image")
	}

	return image, nil
}

func (b *localBuilder) publish(tarFile string, source build.BuildSource) (string, error) {

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
							Name: "fabric8/s2i-java:2.1",
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &v1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: "camel-k-" + source.Identifier.Name + ":" + source.Identifier.Digest,
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

	inConfig := k8sclient.GetKubeConfig()
	config := rest.CopyConfig(inConfig)
	config.GroupVersion = &schema.GroupVersion{
		Group:   "build.openshift.io",
		Version: "v1",
	}
	config.APIPath = "/apis"
	config.AcceptContentTypes = "application/json"
	config.ContentType = "application/json"

	// this gets used for discovery and error handling types
	config.NegotiatedSerializer = basicNegotiatedSerializer{}
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return "", err
	}

	resource, err := ioutil.ReadFile(tarFile)
	if err != nil {
		return "", errors.Wrap(err, "cannot fully read tar file "+tarFile)
	}

	result := restClient.
		Post().
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
	return is.Status.DockerImageRepository + ":" + source.Identifier.Digest, nil
}
