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

package cmd

import (
	"fmt"
	"regexp"
)

// RunConfigOption represents a config option
type RunConfigOption struct {
	ConfigType configOptionType
	Value      string
}

type configOptionType string

const (
	// ConfigOptionTypeConfigmap --
	ConfigOptionTypeConfigmap configOptionType = "configmap"
	// ConfigOptionTypeSecret --
	ConfigOptionTypeSecret configOptionType = "secret"
	// ConfigOptionTypeFile --
	ConfigOptionTypeFile configOptionType = "file"
)

var validConfigRegexp = regexp.MustCompile(`^(configmap|secret|file)\:([\w\.\-\_\:\/]+)$`)

func newRunConfigOption(configType configOptionType, value string) *RunConfigOption {
	return &RunConfigOption{
		ConfigType: configType,
		Value:      value,
	}
}

// ParseConfigOption will parse and return a runConfigOption
func ParseConfigOption(item string) (*RunConfigOption, error) {
	if !validConfigRegexp.MatchString(item) {
		return nil, fmt.Errorf("could not match configuration %s, must match %v regular expression", item, validConfigRegexp)
	}
	// Parse the regexp groups
	groups := validConfigRegexp.FindStringSubmatch(item)
	var cot configOptionType
	switch groups[1] {
	case "configmap":
		cot = ConfigOptionTypeConfigmap
	case "secret":
		cot = ConfigOptionTypeSecret
	case "file":
		cot = ConfigOptionTypeFile
	default:
		// Should never reach this
		return nil, fmt.Errorf("invalid config option type %s", groups[1])
	}
	return newRunConfigOption(cot, groups[2]), nil
}
