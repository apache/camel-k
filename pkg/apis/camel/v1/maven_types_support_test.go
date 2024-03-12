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
	"github.com/stretchr/testify/require"
)

func TestMarshalEmptyProperties(t *testing.T) {
	buf := new(bytes.Buffer)
	e := xml.NewEncoder(buf)

	err := Properties{}.MarshalXML(e, xml.StartElement{
		Name: xml.Name{Local: "root"},
	})

	require.NoError(t, err)

	err = e.Flush()

	require.NoError(t, err)
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

	require.NoError(t, err)

	err = e.Flush()

	require.NoError(t, err)

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

	require.NoError(t, err)

	err = e.Flush()

	require.NoError(t, err)
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

	require.NoError(t, err)

	err = e.Flush()

	require.NoError(t, err)

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

	require.NoError(t, err)

	err = e.Flush()

	require.NoError(t, err)

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

func TestArtifactToString(t *testing.T) {
	a1 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact", a1.GetDependencyID())

	a2 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Type:       "jar",
		Version:    "1.2",
		Classifier: "foo",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:jar:1.2:foo", a2.GetDependencyID())

	a3 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Version:    "1.2",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:1.2", a3.GetDependencyID())

	a4 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Type:       "jar",
		Classifier: "foo",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:jar::foo", a4.GetDependencyID())

	a5 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Classifier: "foo",
		Version:    "1.2",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact::1.2:foo", a5.GetDependencyID())

	a6 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Type:       "bar",
		Version:    "2.2",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:bar:2.2", a6.GetDependencyID())

	a7 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Classifier: "foo",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:::foo", a7.GetDependencyID())

	a8 := MavenArtifact{
		GroupID:    "org.mygroup",
		ArtifactID: "my-artifact",
		Type:       "jar",
	}
	assert.Equal(t, "mvn:org.mygroup:my-artifact:jar", a8.GetDependencyID())
}
