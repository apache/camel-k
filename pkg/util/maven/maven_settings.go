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

package maven

import (
	"encoding/xml"

	"github.com/apache/camel-k/pkg/util"
)

// DefaultMavenRepositories is a comma separated list of default maven repositories
// This variable can be overridden at build time
var DefaultMavenRepositories = "https://repo.maven.apache.org/maven2@id=central"

func (s Settings) MarshalBytes() ([]byte, error) {
	return util.EncodeXML(s)
}

type SettingsOption interface {
	apply(settings *Settings) error
}

func NewSettings(options ...SettingsOption) (Settings, error) {
	settings := Settings{
		XMLName:           xml.Name{Local: "settings"},
		XMLNs:             "http://maven.apache.org/SETTINGS/1.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd",
		Profiles: []Profile{
			{
				ID: "camel-k",
				Activation: Activation{
					ActiveByDefault: true,
				},
			},
		},
	}

	for _, option := range options {
		err := option.apply(&settings)
		if err != nil {
			return Settings{}, err
		}
	}

	return settings, nil
}
