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

import (
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

type PublishStrategyOption struct {
	Name         string
	description  string
	defaultValue string
}

func (o *PublishStrategyOption) ToString() string {
	if o.defaultValue == "" {
		return fmt.Sprintf("%s: %s", o.Name, o.description)
	}
	return fmt.Sprintf("%s: %s, default %s", o.Name, o.description, o.defaultValue)
}

// GetSupportedPublishStrategyOptions provides the supported options for the given strategy. Returns nil if no options are supported.
func GetSupportedPublishStrategyOptions(strategy v1.IntegrationPlatformBuildPublishStrategy) []PublishStrategyOption {
	var supportedOptions map[string]PublishStrategyOption
	switch strategy {
	case v1.IntegrationPlatformBuildPublishStrategyKaniko:
		supportedOptions = kanikoSupportedOptions
	case v1.IntegrationPlatformBuildPublishStrategyBuildah:
		supportedOptions = buildahSupportedOptions
	default:
		return nil
	}
	result := make([]PublishStrategyOption, 0, len(supportedOptions))
	for _, value := range supportedOptions {
		result = append(result, value)
	}
	return result
}

// IsSupportedPublishStrategyOption indicates whether the given option name is supported for the given strategy.
func IsSupportedPublishStrategyOption(strategy v1.IntegrationPlatformBuildPublishStrategy, name string) bool {
	var supportedOption bool
	switch strategy {
	case v1.IntegrationPlatformBuildPublishStrategyKaniko:
		_, supportedOption = kanikoSupportedOptions[name]
	case v1.IntegrationPlatformBuildPublishStrategyBuildah:
		_, supportedOption = buildahSupportedOptions[name]
	default:
		supportedOption = false
	}
	return supportedOption
}
