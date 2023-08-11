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

package resource

import (
	"crypto/sha1" // nolint: gosec
	"fmt"
	"regexp"
	"strings"
)

// Config represents a config option.
type Config struct {
	storageType     StorageType
	contentType     ContentType
	resourceName    string
	resourceKey     string
	destinationPath string
}

// DestinationPath is the location where the resource will be stored on destination.
func (config *Config) DestinationPath() string {
	return config.destinationPath
}

// StorageType is the type of storage used for the configuration.
func (config *Config) StorageType() StorageType {
	return config.storageType
}

// ContentType is the type of content used for the configuration.
func (config *Config) ContentType() ContentType {
	return config.contentType
}

// Name is the name of the resource.
func (config *Config) Name() string {
	return config.resourceName
}

// Key is the key specified for the resource.
func (config *Config) Key() string {
	return config.resourceKey
}

// String represents the unparsed value of the resource.
func (config *Config) String() string {
	s := fmt.Sprintf("%s:%s", config.storageType, config.resourceName)
	if config.resourceKey != "" {
		s = fmt.Sprintf("%s/%s", s, config.resourceKey)
	}
	if config.destinationPath != "" {
		s = fmt.Sprintf("%s@%s", s, config.destinationPath)
	}

	return s
}

// StorageType represent a resource/config type such as configmap, secret or local file.
type StorageType string

const (
	// StorageTypeConfigmap --.
	StorageTypeConfigmap StorageType = "configmap"
	// StorageTypeSecret --.
	StorageTypeSecret StorageType = "secret"
	// StorageTypePVC --.
	StorageTypePVC StorageType = "pvc"
)

// ContentType represent what kind of a content is, either data or purely text configuration.
type ContentType string

const (
	// ContentTypeData can contain binary content, won't be parsed to look for user properties.
	ContentTypeData ContentType = "data"
	// ContentTypeText can't contain binary content, will be parsed to look for user properties.
	ContentTypeText ContentType = "text"
)

var (
	validConfigSecretRegexp = regexp.MustCompile(`^(configmap|secret)\:([\w\.\-\_\:\/@]+)$`)
	validResourceRegexp     = regexp.MustCompile(`^([\w\.\-\_\:]+)(\/([\w\.\-\_\:]+))?(\@([\w\.\-\_\:\/]+))?$`)
)

func newConfig(storageType StorageType, contentType ContentType, value string) *Config {
	rn, mk, mp := parseCMOrSecretValue(value)
	return &Config{
		storageType:     storageType,
		contentType:     contentType,
		resourceName:    rn,
		resourceKey:     mk,
		destinationPath: mp,
	}
}

// ParseFileValue will parse a file resource/config option to return the local path and the
// destination path expected.
func ParseFileValue(value string) (string, string) {
	split := strings.SplitN(value, "@", 2)
	if len(split) == 2 {
		return split[0], split[1]
	}

	return value, ""
}

func parseCMOrSecretValue(value string) (string, string, string) {
	if !validResourceRegexp.MatchString(value) {
		return value, "", ""
	}
	// Must have 3 values
	groups := validResourceRegexp.FindStringSubmatch(value)

	return groups[1], groups[3], groups[5]
}

// ParseResource will parse a resource and return a Config.
func ParseResource(item string) (*Config, error) {
	return parse(item, ContentTypeData)
}

// ParseVolume will parse a volume and return a Config.
func ParseVolume(item string) (*Config, error) {
	configParts := strings.Split(item, ":")

	if len(configParts) != 2 {
		return nil, fmt.Errorf("could not match pvc as %s", item)
	}

	return &Config{
		storageType:     StorageTypePVC,
		resourceName:    configParts[0],
		destinationPath: configParts[1],
	}, nil
}

// ParseConfig will parse a config and return a Config.
func ParseConfig(item string) (*Config, error) {
	return parse(item, ContentTypeText)
}

func parse(item string, contentType ContentType) (*Config, error) {
	var cot StorageType
	var value string
	switch {
	case validConfigSecretRegexp.MatchString(item):
		// parse as secret/configmap
		groups := validConfigSecretRegexp.FindStringSubmatch(item)
		switch groups[1] {
		case "configmap":
			cot = StorageTypeConfigmap
		case "secret":
			cot = StorageTypeSecret
		}
		value = groups[2]
	default:
		return nil, fmt.Errorf("could not match config or secret configuration as %s", item)
	}

	return newConfig(cot, contentType, value), nil
}

func hashFrom(contents ...[]byte) string {
	// SHA1 because we need to limit the length to less than 64 chars
	hash := sha1.New() // nolint: gosec
	for _, c := range contents {
		hash.Write(c)
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}
