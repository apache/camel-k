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
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/kubernetes/customclient"
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

type s2iPublisher struct {
	buffer    chan s2iPublishOperation
	namespace string
}

type s2iPublishOperation struct {
	request   build.Request
	assembled build.AssembledOutput
	packaged  build.PackagedOutput
	output    chan build.PublishedOutput
}

// NewS2IPublisher creates a new publisher doing a Openshift S2I binary build
func NewS2IPublisher(ctx context.Context, namespace string) build.Publisher {
	publisher := s2iPublisher{
		buffer:    make(chan s2iPublishOperation, 100),
		namespace: namespace,
	}
	go publisher.publishCycle(ctx)
	return &publisher
}

func (b *s2iPublisher) Publish(request build.Request, assembled build.AssembledOutput, packaged build.PackagedOutput) <-chan build.PublishedOutput {
	res := make(chan build.PublishedOutput, 1)
	op := s2iPublishOperation{
		request:   request,
		assembled: assembled,
		packaged:  packaged,
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
			res := b.execute(op.request, op.assembled, op.packaged)
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

func (b *s2iPublisher) execute(request build.Request, assembled build.AssembledOutput, packaged build.PackagedOutput) build.PublishedOutput {
	image, err := b.publish(packaged.TarFile, packaged.BaseImage, request)
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
