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

package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JaegerTracingLocator struct {
	allowHeadless bool
}

const (
	jaegerPortName = "http-c-binary-trft"
)

func (loc *JaegerTracingLocator) FindEndpoint(ctx context.Context, c client.Client, l log.Logger,
	e *trait.Environment) (string, error) {
	opts := metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=jaeger,app.kubernetes.io/component=service-collector",
	}
	lst, err := c.CoreV1().Services(e.Integration.Namespace).List(ctx, opts)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, svc := range lst.Items {
		if !loc.allowHeadless && strings.HasSuffix(svc.Name, "-headless") {
			continue
		}

		for _, port := range svc.Spec.Ports {
			if port.Name == jaegerPortName && port.Port > 0 {
				candidates = append(candidates,
					fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/api/traces", svc.Name, svc.Namespace, port.Port))
			}
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 0 {
		for _, endpoint := range candidates {
			l.Infof("Detected Jaeger endpoint at: %s", endpoint)
		}
		return candidates[0], nil
	}
	return "", nil
}

// registering the locator.
func init() {
	TracingLocators = append(TracingLocators, &JaegerTracingLocator{}, &JaegerTracingLocator{allowHeadless: true})
}
