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
	"bytes"
	"encoding/xml"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func NewProject() Project {
	return Project{
		XMLName:           xml.Name{Local: "project"},
		XMLNs:             "http://maven.apache.org/POM/4.0.0",
		XMLNsXsi:          "http://www.w3.org/2001/XMLSchema-instance",
		XsiSchemaLocation: "http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd",
		ModelVersion:      "4.0.0",
	}
}

func NewProjectWithGAV(group string, artifact string, version string) Project {
	p := NewProject()
	p.GroupID = group
	p.ArtifactID = artifact
	p.Version = version
	p.Properties = make(map[string]string)
	p.Properties["project.build.sourceEncoding"] = "UTF-8"

	return p
}

func (p Project) Command(context Context) *Command {
	return &Command{
		context: context,
		project: p,
	}
}

func (p Project) MarshalBytes() ([]byte, error) {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	err := e.Encode(p)
	if err != nil {
		return []byte{}, err
	}

	return w.Bytes(), nil
}

func (p *Project) LookupDependency(dep Dependency) *Dependency {
	for i := range p.Dependencies {
		// Check if the given dependency is already included in the dependency list
		if p.Dependencies[i].GroupID == dep.GroupID && p.Dependencies[i].ArtifactID == dep.ArtifactID {
			return &p.Dependencies[i]
		}
	}

	return nil
}

func (p *Project) ReplaceDependency(dep Dependency) {
	for i, d := range p.Dependencies {
		// Check if the given dependency is already included in the dependency list
		if d.GroupID == dep.GroupID && d.ArtifactID == dep.ArtifactID {
			p.Dependencies[i] = dep

			return
		}
	}
}

// AddDependency adds a dependency to maven's dependencies
func (p *Project) AddDependency(dep Dependency) {
	for _, d := range p.Dependencies {
		// Check if the given dependency is already included in the dependency list
		if d.GroupID == dep.GroupID && d.ArtifactID == dep.ArtifactID {
			return
		}
	}

	p.Dependencies = append(p.Dependencies, dep)
}

// AddDependencies adds dependencies to maven's dependencies
func (p *Project) AddDependencies(deps ...Dependency) {
	for _, d := range deps {
		p.AddDependency(d)
	}
}

// AddDependencyGAV a dependency to maven's dependencies
func (p *Project) AddDependencyGAV(groupID string, artifactID string, version string) {
	p.AddDependency(NewDependency(groupID, artifactID, version))
}

// AddEncodedDependencyGAV a dependency to maven's dependencies
func (p *Project) AddEncodedDependencyGAV(gav string) {
	if d, err := ParseGAV(gav); err == nil {
		// TODO: error handling
		p.AddDependency(d)
	}
}

func (p *Project) AddDependencyExclusion(dep Dependency, exclusion Exclusion) {
	if t := p.LookupDependency(dep); t != nil {
		if t.Exclusions == nil {
			exclusions := make([]Exclusion, 0)
			t.Exclusions = &exclusions
		}

		for _, e := range *t.Exclusions {
			if e.ArtifactID == exclusion.ArtifactID && e.GroupID == exclusion.GroupID {
				return
			}
		}

		*t.Exclusions = append(*t.Exclusions, exclusion)
	}
}

func (p *Project) AddDependencyExclusions(dep Dependency, exclusions ...Exclusion) {
	for _, e := range exclusions {
		p.AddDependencyExclusion(dep, e)
	}
}

type propertiesEntry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func (m Properties) AddAll(properties map[string]string) {
	for k, v := range properties {
		m[k] = v
	}
}

func (m Properties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
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

// NewDependency creates an new dependency from the given GAV
func NewDependency(groupID string, artifactID string, version string) Dependency {
	return Dependency{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		Type:       "",
		Classifier: "",
	}
}

// NewRepository parses the given repository URL and generates the corresponding v1.Repository.
//
// The repository can be customized by appending @param to the repository
// URL, e.g.:
//
//     http://my-nexus:8081/repository/publicc@id=my-repo@snapshots
//
// That enables snapshots and sets the repository id to `my-repo`
func NewRepository(repo string) v1.Repository {
	r := v1.Repository{
		URL: repo,
		Releases: v1.RepositoryPolicy{
			Enabled:        true,
			ChecksumPolicy: "fail",
		},
		Snapshots: v1.RepositoryPolicy{
			Enabled:        false,
			ChecksumPolicy: "fail",
		},
	}

	if idx := strings.Index(repo, "@"); idx != -1 {
		r.URL = repo[:idx]

		for _, attribute := range strings.Split(repo[idx+1:], "@") {
			switch {
			case attribute == "snapshots":
				r.Snapshots.Enabled = true
			case attribute == "noreleases":
				r.Releases.Enabled = false
			case strings.HasPrefix(attribute, "id="):
				r.ID = attribute[3:]
			case strings.HasPrefix(attribute, "name="):
				r.Name = attribute[5:]
			case strings.HasPrefix(attribute, "checksumpolicy="):
				r.Snapshots.ChecksumPolicy = attribute[15:]
				r.Releases.ChecksumPolicy = attribute[15:]
			}
		}
	}

	return r
}

func NewMirror(repo string) Mirror {
	m := Mirror{}
	if idx := strings.Index(repo, "@"); idx != -1 {
		m.URL = repo[:idx]

		for _, attribute := range strings.Split(repo[idx+1:], "@") {
			switch {
			case strings.HasPrefix(attribute, "mirrorOf="):
				m.MirrorOf = attribute[9:]
			case strings.HasPrefix(attribute, "id="):
				m.ID = attribute[3:]
			case strings.HasPrefix(attribute, "name="):
				m.Name = attribute[5:]
			}
		}
	}
	return m
}
