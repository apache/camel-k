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

package v1

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalEmptyProperties(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	err := Properties{}.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	assert.NoError(t, err)

	err = e.Flush()

	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
}

func TestMarshalProperties(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	properties := Properties{}
	properties.Add("v1", "foo")
	properties.Add("v2", "bar")

	err := properties.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	assert.NoError(t, err)

	err = e.Flush()

	assert.NoError(t, err)

	result := buf.String()
	assert.True(t, strings.HasPrefix(result, "<root>"))
	assert.True(t, strings.HasSuffix(result, "</root>"))
	assert.Contains(t, result, "<v1>foo</v1>")
	assert.Contains(t, result, "<v2>bar</v2>")
}

func TestMarshalEmptyPluginProperties(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	err := PluginProperties{}.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	assert.NoError(t, err)

	err = e.Flush()

	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
}

func TestMarshalPluginProperties(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	properties := PluginProperties{}
	properties.Add("v1", "foo")
	properties.Add("v2", "bar")

	err := properties.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	assert.NoError(t, err)

	err = e.Flush()

	assert.NoError(t, err)

	result := buf.String()
	assert.True(t, strings.HasPrefix(result, "<root>"))
	assert.True(t, strings.HasSuffix(result, "</root>"))
	assert.Contains(t, result, "<v1>foo</v1>")
	assert.Contains(t, result, "<v2>bar</v2>")
}

func TestMarshalPluginPropertiesWithNestedProps(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	properties := PluginProperties{}
	properties.Add("v1", "foo")
	properties.AddProperties("props", map[string]string{
		"prop1": "foo",
		"prop2": "baz",
	})
	properties.Add("v2", "bar")

	err := properties.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	assert.NoError(t, err)

	err = e.Flush()

	assert.NoError(t, err)

	result := buf.String()
	assert.True(t, strings.HasPrefix(result, "<root>"))
	assert.True(t, strings.HasSuffix(result, "</root>"))
	assert.Contains(t, result, "<v1>foo</v1>")
	assert.Contains(t, result, "<props>")
	assert.Contains(t, result, "</props>")
	assert.Contains(t, result, "<prop1>foo</prop1>")
	assert.Contains(t, result, "<prop2>baz</prop2>")
	assert.Contains(t, result, "<v2>bar</v2>")
}
