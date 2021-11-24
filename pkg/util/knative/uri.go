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

package knative

import (
	"fmt"
	"regexp"

	knativev1 "github.com/apache/camel-k/pkg/apis/camel/v1/knative"
	uriutils "github.com/apache/camel-k/pkg/util/uri"
	v1 "k8s.io/api/core/v1"
)

var (
	uriRegexp       = regexp.MustCompile(`^knative:[/]*(channel|endpoint|event)(?:[?].*|$|/([A-Za-z0-9.-]+)(?:[/?].*|$))`)
	plainNameRegexp = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)
)

const (
	paramAPIVersion = "apiVersion"
	paramKind       = "kind"
	paramBrokerName = "name"
)

// FilterURIs returns all Knative URIs of the given type from a slice.
func FilterURIs(uris []string, kind knativev1.CamelServiceType) []string {
	res := make([]string, 0)
	for _, uri := range uris {
		if isKnativeURI(kind, uri) {
			res = append(res, uri)
		}
	}
	return res
}

// NormalizeToURI produces a Knative uri of the given service type if the argument is a plain string.
func NormalizeToURI(kind knativev1.CamelServiceType, uriOrString string) string {
	if plainNameRegexp.MatchString(uriOrString) {
		return fmt.Sprintf("knative://%s/%s", string(kind), uriOrString)
	}
	return uriOrString
}

// ExtractObjectReference returns a reference to the object described in the Knative URI.
func ExtractObjectReference(uri string) (v1.ObjectReference, error) {
	if isKnativeURI(knativev1.CamelServiceTypeEvent, uri) {
		name := uriutils.GetQueryParameter(uri, paramBrokerName)
		if name == "" {
			name = "default"
		}
		apiVersion := uriutils.GetQueryParameter(uri, paramAPIVersion)
		return v1.ObjectReference{
			Name:       name,
			APIVersion: apiVersion,
			Kind:       "Broker",
		}, nil
	}
	name := matchOrEmpty(uriRegexp, 2, uri)
	if name == "" {
		return v1.ObjectReference{}, fmt.Errorf("cannot find name in uri %s", uri)
	}
	apiVersion := uriutils.GetQueryParameter(uri, paramAPIVersion)
	kind := uriutils.GetQueryParameter(uri, paramKind)
	return v1.ObjectReference{
		Name:       name,
		APIVersion: apiVersion,
		Kind:       kind,
	}, nil
}

// ExtractEventType extract the eventType from a event URI.
func ExtractEventType(uri string) string {
	return matchOrEmpty(uriRegexp, 2, uri)
}

func matchOrEmpty(reg *regexp.Regexp, index int, str string) string {
	match := reg.FindStringSubmatch(str)
	if len(match) > index {
		return match[index]
	}
	return ""
}

func isKnativeURI(kind knativev1.CamelServiceType, uri string) bool {
	match := uriRegexp.FindStringSubmatch(uri)
	if len(match) == 3 && match[1] == string(kind) {
		return true
	}
	return false
}
