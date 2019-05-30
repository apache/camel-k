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
	"testing"

	"github.com/apache/camel-k/pkg/util"
	"github.com/stretchr/testify/assert"
)

const expectedSettings = `<?xml version="1.0" encoding="UTF-8"?>
<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">
  <proxies>
    <proxy>
      <active>false</active>
      <port>8128</port>
      <id>local_http</id>
      <protocol>http</protocol>
      <host>localhost</host>
    </proxy>
  </proxies>
</settings>`

func TestSettingsGeneration(t *testing.T) {
	settings := NewSettings()
	settings.Proxies = []Proxy{
		{
			ID:       "local_http",
			Protocol: "http",
			Host:     "localhost",
			Port:     8128,
		},
	}

	content, err := util.EncodeXML(settings)

	assert.Nil(t, err)
	assert.NotNil(t, settings)

	assert.Equal(t, expectedSettings, content)
}
