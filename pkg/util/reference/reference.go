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

package reference

import (
	"fmt"
	"regexp"
	"unicode"

	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	KameletPrefix = "kamelet:"
)

var (
	simpleNameRegexp = regexp.MustCompile(`^(?:(?P<namespace>[a-z0-9-.]+)/)?(?P<name>[a-z0-9-.]+)$`)
	fullNameRegexp   = regexp.MustCompile(`^(?:(?P<apiVersion>(?:[a-z0-9-.]+/)?(?:[a-z0-9-.]+)):)?(?P<kind>[A-Za-z0-9-.]+):(?:(?P<namespace>[a-z0-9-.]+)/)?(?P<name>[a-z0-9-.]+)$`)

	templates = map[string]corev1.ObjectReference{
		"kamelet": corev1.ObjectReference{
			Kind:       "Kamelet",
			APIVersion: camelv1alpha1.SchemeGroupVersion.String(),
		},
		"channel": corev1.ObjectReference{
			Kind:       "Channel",
			APIVersion: messagingv1.SchemeGroupVersion.String(),
		},
		"broker": corev1.ObjectReference{
			Kind:       "Broker",
			APIVersion: eventingv1.SchemeGroupVersion.String(),
		},
		"ksvc": corev1.ObjectReference{
			Kind:       "Service",
			APIVersion: servingv1.SchemeGroupVersion.String(),
		},
	}
)

type Converter struct {
	defaultPrefix string
}

func NewConverter(defaultPrefix string) *Converter {
	return &Converter{
		defaultPrefix: defaultPrefix,
	}
}

func (c *Converter) FromString(str string) (corev1.ObjectReference, error) {
	ref, err := c.simpleDecodeString(str)
	if err != nil {
		return ref, err
	}
	c.expandReference(&ref)

	if ref.Kind == "" || !unicode.IsUpper([]rune(ref.Kind)[0]) {
		return corev1.ObjectReference{}, fmt.Errorf("invalid kind: %q", ref.Kind)
	}
	return ref, nil
}

func (c *Converter) expandReference(ref *corev1.ObjectReference) {
	if template, ok := templates[ref.Kind]; ok {
		if template.Kind != "" {
			ref.Kind = template.Kind
		}
		if ref.APIVersion == "" && template.APIVersion != "" {
			ref.APIVersion = template.APIVersion
		}
	}
}

func (c *Converter) simpleDecodeString(str string) (corev1.ObjectReference, error) {
	fullName := str
	if simpleNameRegexp.MatchString(str) {
		fullName = c.defaultPrefix + str
	}

	if fullNameRegexp.MatchString(fullName) {
		groupNames := fullNameRegexp.SubexpNames()
		ref := corev1.ObjectReference{}
		for _, match := range fullNameRegexp.FindAllStringSubmatch(fullName, -1) {
			for idx, text := range match {
				groupName := groupNames[idx]
				switch groupName {
				case "apiVersion":
					ref.APIVersion = text
				case "namespace":
					ref.Namespace = text
				case "kind":
					ref.Kind = text
				case "name":
					ref.Name = text
				}
			}
		}
		return ref, nil
	}
	if c.defaultPrefix != "" {
		return corev1.ObjectReference{}, fmt.Errorf(`name %q does not match either "[[apigroup/]version:]kind:[namespace/]name" or "[namespace/]name"`, str)
	}
	return corev1.ObjectReference{}, fmt.Errorf(`name %q does not match format "[[apigroup/]version:]kind:[namespace/]name"`, str)
}

func (c *Converter) ToString(ref corev1.ObjectReference) (string, error) {
	return "", nil
}
