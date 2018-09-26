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

package publish

import (
	"context"
	"io/ioutil"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/tar"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	artifactDirPrefix = "s2i-"
	baseImage         = "fabric8/s2i-java:2.3"
)

type s2iPublisher struct {
	buffer    chan publishOperation
	namespace string
	uploadedArtifactsSelector
}

type publishOperation struct {
	request   build.Request
	assembled build.AssembledOutput
	output    chan build.PublishedOutput
}

type uploadedArtifactsSelector func([]build.ClasspathEntry) (string, []build.ClasspathEntry, error)

// NewS2IPublisher creates a new publisher doing a Openshift S2I binary build
func NewS2IPublisher(ctx context.Context, namespace string) build.Publisher {
	identitySelector := func(entries []build.ClasspathEntry) (string, []build.ClasspathEntry, error) {
		return baseImage, entries, nil
	}
	return newS2IPublisher(ctx, namespace, identitySelector)
}

// NewS2IPublisher creates a new publisher doing a Openshift S2I binary build
func newS2IPublisher(ctx context.Context, namespace string, uploadedArtifactsSelector uploadedArtifactsSelector) *s2iPublisher {
	publisher := s2iPublisher{
		buffer:                    make(chan publishOperation, 100),
		namespace:                 namespace,
		uploadedArtifactsSelector: uploadedArtifactsSelector,
	}
	go publisher.publishCycle(ctx)
	return &publisher
}

func (b *s2iPublisher) Publish(request build.Request, assembled build.AssembledOutput) <-chan build.PublishedOutput {
	res := make(chan build.PublishedOutput, 1)
	op := publishOperation{
		request:   request,
		assembled: assembled,
		output:    res,
	}
	b.buffer <- op
	return res
}

func (b *s2iPublisher) publishCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			b.buffer = nil
			return
		case op := <-b.buffer:
			now := time.Now()
			logrus.Info("Starting a new image publication")
			res := b.execute(op.request, op.assembled)
			elapsed := time.Now().Sub(now)

			if res.Error != nil {
				logrus.Error("Error during publication (total time ", elapsed.Seconds(), " seconds): ", res.Error)
			} else {
				logrus.Info("Publication completed in ", elapsed.Seconds(), " seconds")
			}

			op.output <- res
		}
	}
}

func (b *s2iPublisher) execute(request build.Request, assembled build.AssembledOutput) build.PublishedOutput {
	baseImageName, selectedArtifacts, err := b.uploadedArtifactsSelector(assembled.Classpath)
	if err != nil {
		return build.PublishedOutput{Error: err}
	}

	tarFile, err := b.createTar(assembled, selectedArtifacts)
	if err != nil {
		return build.PublishedOutput{Error: err}
	}

	image, err := b.publish(tarFile, baseImageName, request)
	if err != nil {
		return build.PublishedOutput{Error: err}
	}

	return build.PublishedOutput{Image: image}
}

func (b *s2iPublisher) publish(tarFile string, imageName string, source build.Request) (string, error) {

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
							Name: imageName,
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

func (b *s2iPublisher) createTar(assembled build.AssembledOutput, selectedArtifacts []build.ClasspathEntry) (string, error) {
	artifactDir, err := ioutil.TempDir("", artifactDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for s2i artifacts")
	}

	tarFileName := path.Join(artifactDir, "occi.tar")
	tarAppender, err := tar.NewAppender(tarFileName)
	if err != nil {
		return "", err
	}
	defer tarAppender.Close()

	tarDir := "dependencies/"
	for _, entry := range selectedArtifacts {
		gav, err := maven.ParseGAV(entry.ID)
		if err != nil {
			return "", nil
		}

		tarPath := path.Join(tarDir, gav.GroupID)
		_, err = tarAppender.AddFile(entry.Location, tarPath)
		if err != nil {
			return "", err
		}
	}

	cp := ""
	for _, entry := range assembled.Classpath {
		gav, err := maven.ParseGAV(entry.ID)
		if err != nil {
			return "", nil
		}
		tarPath := path.Join(tarDir, gav.GroupID)
		_, fileName := path.Split(entry.Location)
		fileName = path.Join(tarPath, fileName)
		cp += fileName + "\n"
	}

	err = tarAppender.AppendData([]byte(cp), "classpath")
	if err != nil {
		return "", err
	}

	return tarFileName, nil
}
