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

package scalers

import (
	"fmt"
	"strings"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
)

// AWS2SQSScaler maps the Camel aws2-sqs consumer to the KEDA aws-sqs-queue scaler.
type AWS2SQSScaler struct{}

func init() {
	keda.Register(&AWS2SQSScaler{})
}

func (s *AWS2SQSScaler) Component() string {
	return "aws2-sqs"
}

// Map derives the KEDA queueURL and awsRegion from the endpoint URI.
// An explicit queueUrl parameter wins. Otherwise the URL is built from an
// SQS queue ARN, or from a queue name together with the region and
// queueOwnerAWSAccountId parameters. When no full URL can be built, the
// plain queue name is used, which KEDA accepts when it is unambiguous.
func (s *AWS2SQSScaler) Map(pathValue string, params map[string]string) (string, map[string]string) {
	metadata := make(map[string]string)

	region := params["region"]
	pathValue = strings.TrimPrefix(pathValue, "//")
	queueName := pathValue
	account := params["queueOwnerAWSAccountId"]

	// arn:aws:sqs:<region>:<account>:<queue>
	if parts := strings.Split(pathValue, ":"); len(parts) == 6 && parts[0] == "arn" && parts[2] == "sqs" {
		if region == "" {
			region = parts[3]
		}
		account = parts[4]
		queueName = parts[5]
	}

	if region != "" {
		metadata["awsRegion"] = region
	}

	switch {
	case params["queueUrl"] != "":
		metadata["queueURL"] = params["queueUrl"]
	case queueName != "" && account != "" && region != "":
		metadata["queueURL"] = fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", region, account, queueName)
	case queueName != "":
		metadata["queueURL"] = queueName
	}

	if params["overrideEndpoint"] == "true" && params["uriEndpointOverride"] != "" {
		metadata["awsEndpoint"] = params["uriEndpointOverride"]
	}

	return "aws-sqs-queue", metadata
}
