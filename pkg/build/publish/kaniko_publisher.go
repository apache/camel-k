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
	tarutils "archive/tar"
	"context"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kanikoPublisher struct {
	buffer    chan kanikoPublishOperation
	namespace string
	registry  string
}

type kanikoPublishOperation struct {
	request   build.Request
	assembled build.AssembledOutput
	packaged  build.PackagedOutput
	output    chan build.PublishedOutput
}

// NewKanikoPublisher creates a new publisher doing a Kaniko image push
func NewKanikoPublisher(ctx context.Context, namespace string, registry string) build.Publisher {
	publisher := kanikoPublisher{
		buffer:    make(chan kanikoPublishOperation, 100),
		namespace: namespace,
		registry:  registry,
	}
	go publisher.publishCycle(ctx)
	return &publisher
}

func (b *kanikoPublisher) Publish(request build.Request, assembled build.AssembledOutput, packaged build.PackagedOutput) <-chan build.PublishedOutput {
	res := make(chan build.PublishedOutput, 1)
	op := kanikoPublishOperation{
		request:   request,
		assembled: assembled,
		packaged:  packaged,
		output:    res,
	}
	b.buffer <- op
	return res
}

func (b *kanikoPublisher) publishCycle(ctx context.Context) {
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

func (b *kanikoPublisher) execute(request build.Request, assembled build.AssembledOutput, packaged build.PackagedOutput) build.PublishedOutput {
	image, err := b.publish(packaged.TarFile, packaged.BaseImage, request)
	if err != nil {
		return build.PublishedOutput{Error: err}
	}

	return build.PublishedOutput{Image: image}
}

func (b *kanikoPublisher) publish(tarFile string, baseImageName string, source build.Request) (string, error) {
	image := b.registry + "/" + b.namespace + "/camel-k-" + source.Identifier.Name + ":" + source.Identifier.Qualifier
	contextDir, err := b.prepareContext(tarFile, baseImageName)
	if err != nil {
		return "", err
	}
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: b.namespace,
			Name:      "camel-k-" + source.Identifier.Name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "kaniko",
					Image: "gcr.io/kaniko-project/executor:latest",
					Args: []string{
						"--dockerfile=Dockerfile",
						"--context=" + contextDir,
						"--destination=" + image,
						"--insecure",
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "camel-k-builder",
							MountPath: "/workspace",
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name: "camel-k-builder",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: "camel-k-builder",
							ReadOnly:  true,
						},
					},
				},
			},
		},
	}

	sdk.Delete(&pod)
	err = sdk.Create(&pod)
	if err != nil {
		return "", errors.Wrap(err, "cannot create kaniko builder pod")
	}

	err = kubernetes.WaitCondition(&pod, func(obj interface{}) (bool, error) {
		if val, ok := obj.(*v1.Pod); ok {
			if val.Status.Phase == v1.PodSucceeded {
				return true, nil
			} else if val.Status.Phase == v1.PodFailed {
				return false, errors.New("build failed")
			}
		}
		return false, nil
	}, 5*time.Minute)

	if err != nil {
		return "", err
	}

	return image, nil
}

func (b *kanikoPublisher) prepareContext(tarName string, baseImage string) (string, error) {
	baseDir, _ := path.Split(tarName)
	contextDir := path.Join(baseDir, "context")
	if err := b.unTar(tarName, contextDir); err != nil {
		return "", err
	}

	dockerFileContent := []byte(`
		FROM ` + baseImage + `
		ADD . /deployments
	`)
	if err := ioutil.WriteFile(path.Join(contextDir, "Dockerfile"), dockerFileContent, 0777); err != nil {
		return "", err
	}
	return contextDir, nil
}

func (b *kanikoPublisher) unTar(tarName string, dir string) error {
	file, err := os.Open(tarName)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := tarutils.NewReader(file)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		targetName := path.Join(dir, header.Name)
		targetDir, _ := path.Split(targetName)
		if err := os.MkdirAll(targetDir, 0777); err != nil {
			return err
		}
		buffer, err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(targetName, buffer, os.FileMode(header.Mode)); err != nil {
			return err
		}
	}
	return nil
}
