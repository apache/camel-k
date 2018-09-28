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

const (
	s2iRootImage = "fabric8/s2i-java:2.3"
	//javaRootImage = "fabric8/java-jboss-openjdk8-jdk:1.5.1"
	javaRootImage = "fabric8/java-alpine-openjdk8-jdk:1.5.1"
)

// NewS2IStandardPackager creates a standard packager for S2I builds
func NewS2IStandardPackager(ctx context.Context) build.Packager {
	return newBasePackager(ctx, s2iRootImage)
}

// NewS2IIncrementalPackager creates a incremental packager for S2I builds
func NewS2IIncrementalPackager(ctx context.Context, lister PublishedImagesLister) build.Packager {
	return newIncrementalPackager(ctx, lister, s2iRootImage)
}

// NewJavaStandardPackager creates a standard packager for Java Docker builds
func NewJavaStandardPackager(ctx context.Context) build.Packager {
	return newBasePackager(ctx, javaRootImage)
}

// NewJavaIncrementalPackager creates a incremental packager for Java Docker builds
func NewJavaIncrementalPackager(ctx context.Context, lister PublishedImagesLister) build.Packager {
	return newIncrementalPackager(ctx, lister, javaRootImage)
}
