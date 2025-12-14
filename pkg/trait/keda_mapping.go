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

package trait

import (
	"net/url"
	"strings"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/uri"
)

type CamelToKedaMapping struct {
	KedaScalerType string
	// Maps Camel URI Param names to KEDA metadata keys.
	ParameterMap map[string]string
	// Params that are taken from URI path, (component:path?queryParams)
	PathParamName string
}

// camelToKedaMappings maps Camel component URI parameters to KEDA scaler metadata.
// Only components available in Camel-K catalog (pkg/resources/resources/camel-catalog-*.yaml)
// that have corresponding KEDA scalers are included.
var camelToKedaMappings = map[string]CamelToKedaMapping{
	"kafka": {
		KedaScalerType: "kafka",
		PathParamName:  "topic",
		ParameterMap: map[string]string{
			"brokers": "bootstrapServers",
			"groupId": "consumerGroup",
		},
	},
	"aws2-sqs": {
		KedaScalerType: "aws-sqs-queue",
		PathParamName:  "queueURL",
		ParameterMap: map[string]string{
			"region": "awsRegion",
		},
	},
	"spring-rabbitmq": {
		KedaScalerType: "rabbitmq",
		PathParamName:  "",
		ParameterMap: map[string]string{
			"queues":    "queueName",
			"addresses": "host",
		},
	},
}

// parseComponentURI extracts the component scheme, path value, and query parameters from a Camel URI.
func parseComponentURI(rawURI string) (string, string, map[string]string, error) {
	scheme := uri.GetComponent(rawURI)
	if scheme == "" {
		return "", "", nil, nil
	}

	params := make(map[string]string)

	// extract path
	remainder := strings.TrimPrefix(rawURI, scheme+":")
	var pathValue string
	if idx := strings.Index(remainder, "?"); idx >= 0 {
		pathValue = remainder[:idx]
		queryString := remainder[idx+1:]

		values, parseErr := url.ParseQuery(queryString)
		if parseErr != nil {
			return "", "", nil, parseErr
		}
		for k, v := range values {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		pathValue = remainder
	}

	return scheme, pathValue, params, nil
}

// mapToKedaTrigger converts a Camel URI to a KEDA trigger if the component is supported.
func mapToKedaTrigger(rawURI string) (*traitv1.KedaTrigger, error) {
	scheme, pathValue, params, err := parseComponentURI(rawURI)
	if err != nil {
		return nil, err
	}

	mapping, found := camelToKedaMappings[scheme]
	if scheme == "" || !found {
		return nil, nil // no trigger for this URI
	}

	metadata := make(map[string]string)

	if mapping.PathParamName != "" && pathValue != "" {
		metadata[mapping.PathParamName] = pathValue
	}

	for camelParam, kedaParam := range mapping.ParameterMap {
		if val, ok := params[camelParam]; ok {
			metadata[kedaParam] = val
		}
	}

	return &traitv1.KedaTrigger{
		Type:     mapping.KedaScalerType,
		Metadata: metadata,
	}, nil
}
