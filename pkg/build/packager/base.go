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

package packager

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/apache/camel-k/pkg/util/tar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	sharedDir         = "/workspace"
	artifactDirPrefix = "layer-"
)

type commonPackager struct {
	buffer chan packageOperation
	uploadedArtifactsSelector
}

type packageOperation struct {
	request   build.Request
	assembled build.AssembledOutput
	output    chan build.PackagedOutput
}

type uploadedArtifactsSelector func([]build.ClasspathEntry) (string, []build.ClasspathEntry, error)

func newBasePackager(ctx context.Context, rootImage string) build.Packager {
	identitySelector := func(entries []build.ClasspathEntry) (string, []build.ClasspathEntry, error) {
		return rootImage, entries, nil
	}
	return newBasePackagerWithSelector(ctx, identitySelector)
}

func newBasePackagerWithSelector(ctx context.Context, uploadedArtifactsSelector uploadedArtifactsSelector) *commonPackager {
	pack := commonPackager{
		buffer: make(chan packageOperation, 100),
		uploadedArtifactsSelector: uploadedArtifactsSelector,
	}
	go pack.packageCycle(ctx)
	return &pack
}

func (b *commonPackager) Package(request build.Request, assembled build.AssembledOutput) <-chan build.PackagedOutput {
	res := make(chan build.PackagedOutput, 1)
	op := packageOperation{
		request:   request,
		assembled: assembled,
		output:    res,
	}
	b.buffer <- op
	return res
}

func (b *commonPackager) Cleanup(output build.PackagedOutput) {
	parentDir, _ := path.Split(output.TarFile)
	err := os.RemoveAll(parentDir)
	if err != nil {
		logrus.Warn("Could not remove temporary directory ", parentDir)
	}
}

func (b *commonPackager) packageCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			b.buffer = nil
			return
		case op := <-b.buffer:
			now := time.Now()
			logrus.Info("Starting a new image packaging")
			res := b.execute(op.request, op.assembled)
			elapsed := time.Now().Sub(now)

			if res.Error != nil {
				logrus.Error("Error during packaging (total time ", elapsed.Seconds(), " seconds): ", res.Error)
			} else {
				logrus.Info("Packaging completed in ", elapsed.Seconds(), " seconds")
			}

			op.output <- res
		}
	}
}

func (b *commonPackager) execute(request build.Request, assembled build.AssembledOutput) build.PackagedOutput {
	baseImageName, selectedArtifacts, err := b.uploadedArtifactsSelector(assembled.Classpath)
	if err != nil {
		return build.PackagedOutput{Error: err}
	}

	tarFile, err := b.createTar(assembled, selectedArtifacts)
	if err != nil {
		return build.PackagedOutput{Error: err}
	}

	return build.PackagedOutput{
		BaseImage: baseImageName,
		TarFile:   tarFile,
	}
}

func (b *commonPackager) createTar(assembled build.AssembledOutput, selectedArtifacts []build.ClasspathEntry) (string, error) {
	artifactDir, err := ioutil.TempDir(sharedDir, artifactDirPrefix)
	if err != nil {
		return "", errors.Wrap(err, "could not create temporary dir for packaged artifacts")
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
