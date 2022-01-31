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

import "encoding/xml"

// Repository --
type Repository struct {
	ID        string           `xml:"id" json:"id"`
	Name      string           `xml:"name,omitempty" json:"name,omitempty"`
	URL       string           `xml:"url" json:"url"`
	Snapshots RepositoryPolicy `xml:"snapshots,omitempty" json:"snapshots,omitempty"`
	Releases  RepositoryPolicy `xml:"releases,omitempty" json:"releases,omitempty"`
}

// RepositoryPolicy --
type RepositoryPolicy struct {
	Enabled        bool   `xml:"enabled" json:"enabled"`
	UpdatePolicy   string `xml:"updatePolicy,omitempty" json:"updatePolicy,omitempty"`
	ChecksumPolicy string `xml:"checksumPolicy,omitempty" json:"checksumPolicy,omitempty"`
}

type Server struct {
	XMLName       xml.Name   `xml:"server"`
	ID            string     `xml:"id,omitempty" json:"id,omitempty"`
	Username      string     `xml:"username,omitempty" json:"username,omitempty"`
	Password      string     `xml:"password,omitempty" json:"password,omitempty"`
	Configuration Properties `xml:"configuration,omitempty" json:"configuration,omitempty"`
}

type Properties map[string]string



type propertiesEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func (m v1.Properties) AddAll(properties map[string]string) {
	for k, v := range properties {
		m[k] = v
	}
}

func (m v1.Properties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(m) == 0 {
		return nil
	}

	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	for k, v := range m {
		if err := e.Encode(propertiesEntry{XMLName: xml.Name{Local: k}, Value: v}); err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}
