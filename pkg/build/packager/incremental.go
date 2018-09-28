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

	"github.com/apache/camel-k/pkg/build"
)

type incrementalPackager struct {
	commonPackager *commonPackager
	lister         PublishedImagesLister
	rootImage      string
}

// newIncrementalPackager creates a new packager that is able to create a layer on top of a existing image
func newIncrementalPackager(ctx context.Context, lister PublishedImagesLister, rootImage string) build.Packager {
	layeredPackager := incrementalPackager{
		lister:    lister,
		rootImage: rootImage,
	}
	layeredPackager.commonPackager = newBasePackagerWithSelector(ctx, layeredPackager.selectArtifactsToUpload)
	return &layeredPackager
}

func (p *incrementalPackager) Package(req build.Request, assembled build.AssembledOutput) <-chan build.PackagedOutput {
	return p.commonPackager.Package(req, assembled)
}

func (p *incrementalPackager) Cleanup(output build.PackagedOutput) {
	p.commonPackager.Cleanup(output)
}

func (p *incrementalPackager) selectArtifactsToUpload(entries []build.ClasspathEntry) (string, []build.ClasspathEntry, error) {
	images, err := p.lister.ListPublishedImages()
	if err != nil {
		return "", nil, err
	}

	bestImage, commonLibs := p.findBestImage(images, entries)
	if bestImage != nil {
		selectedClasspath := make([]build.ClasspathEntry, 0)
		for _, entry := range entries {
			if _, isCommon := commonLibs[entry.ID]; !isCommon {
				selectedClasspath = append(selectedClasspath, entry)
			}
		}

		return bestImage.Image, selectedClasspath, nil
	}

	// return default selection
	return p.rootImage, entries, nil
}

func (p *incrementalPackager) findBestImage(images []PublishedImage, entries []build.ClasspathEntry) (*PublishedImage, map[string]bool) {
	if len(images) == 0 {
		return nil, nil
	}
	requiredLibs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		requiredLibs[entry.ID] = true
	}

	var bestImage PublishedImage
	bestImageCommonLibs := make(map[string]bool, 0)
	bestImageSurplusLibs := 0
	for _, image := range images {
		common := make(map[string]bool)
		for _, id := range image.Classpath {
			if _, ok := requiredLibs[id]; ok {
				common[id] = true
			}
		}
		numCommonLibs := len(common)
		surplus := len(image.Classpath) - numCommonLibs
		if surplus >= numCommonLibs/3 {
			// Heuristic approach: if there are too many unrelated libraries, just use the base image
			continue
		}

		if numCommonLibs > len(bestImageCommonLibs) || (numCommonLibs == len(bestImageCommonLibs) && surplus < bestImageSurplusLibs) {
			bestImage = image
			bestImageCommonLibs = common
			bestImageSurplusLibs = surplus
		}
	}

	return &bestImage, bestImageCommonLibs
}
