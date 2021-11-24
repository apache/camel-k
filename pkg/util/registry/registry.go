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

package registry

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var knownServersByRegistry = map[string]string{
	"docker.io": "https://index.docker.io/v1/",
}

// Auth contains basic information for authenticating against a container registry.
type Auth struct {
	Server   string
	Username string
	Password string

	// additional information
	Registry string
}

type dockerConfigList struct {
	Auths map[string]dockerConfig `json:"auths,omitempty"`
}

type dockerConfig struct {
	Auth string `json:"auth,omitempty"`
}

// IsSet returns if information has been set on the object.
func (a Auth) IsSet() bool {
	return a.Server != "" ||
		a.Username != "" ||
		a.Password != ""
}

// validate checks if all fields are populated correctly.
func (a Auth) validate() error {
	if a.getActualServer() == "" || a.Username == "" {
		return errors.New("not enough information to generate a registry authentication file")
	}

	return nil
}

// GenerateDockerConfig generates a Docker compatible config.json file.
func (a Auth) GenerateDockerConfig() ([]byte, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	content := a.generateDockerConfigObject()

	return json.Marshal(content)
}

func (a Auth) generateDockerConfigObject() dockerConfigList {
	return dockerConfigList{
		map[string]dockerConfig{
			a.getActualServer(): {
				a.encodedCredentials(),
			},
		},
	}
}

func (a Auth) getActualServer() string {
	if a.Server != "" {
		return a.Server
	}
	if p, ok := knownServersByRegistry[a.Registry]; ok {
		return p
	}

	return a.Registry
}

func (a Auth) encodedCredentials() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", a.Username, a.Password)))
}
