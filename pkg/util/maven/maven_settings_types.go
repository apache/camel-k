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

import "encoding/xml"

// Settings represent a maven settings
type Settings struct {
	XMLName           xml.Name
	XMLNs             string  `xml:"xmlns,attr"`
	XMLNsXsi          string  `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation string  `xml:"xsi:schemaLocation,attr"`
	Proxies           []Proxy `xml:"proxies>proxy,omitempty"`
}

// Proxy --
type Proxy struct {
	Active        bool   `xml:"active"`
	Port          int32  `xml:"port"`
	ID            string `xml:"id"`
	Protocol      string `xml:"protocol"`
	Host          string `xml:"host"`
	NonProxyHosts string `xml:"nonProxyHosts,omitempty"`
	Username      string `xml:"username,omitempty"`
	Password      string `xml:"password,omitempty"`
}
