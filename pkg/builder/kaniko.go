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

package builder

// KanikoCacheDir is the cache directory for Kaniko builds (mounted into the Kaniko pod).
const KanikoCacheDir = "/kaniko/cache"
const KanikoPVCName = "KanikoPersistentVolumeClaim"
const KanikoBuildCacheEnabled = "KanikoBuildCacheEnabled"
const KanikoExecutorImage = "KanikoExecutorImage"
const KanikoWarmerImage = "KanikoWarmerImage"
const KanikoDefaultExecutorImageName = "gcr.io/kaniko-project/executor"
const KanikoDefaultWarmerImageName = "gcr.io/kaniko-project/warmer"

var kanikoSupportedOptions = map[string]PublishStrategyOption{
	KanikoPVCName: {
		Name:        KanikoPVCName,
		description: "The name of the PersistentVolumeClaim",
	},
	KanikoBuildCacheEnabled: {
		Name:         KanikoBuildCacheEnabled,
		description:  "To enable or disable the Kaniko cache",
		defaultValue: "false",
	},
	KanikoExecutorImage: {
		Name:         KanikoExecutorImage,
		description:  "The docker image of the Kaniko executor",
		defaultValue: KanikoDefaultExecutorImageName,
	},
	KanikoWarmerImage: {
		Name:         KanikoWarmerImage,
		description:  "The docker image of the Kaniko warmer",
		defaultValue: KanikoDefaultWarmerImageName,
	},
}
