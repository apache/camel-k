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

package generators

import (
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util"

	"gopkg.in/yaml.v2"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"
)

const traitFile = "traits.yaml"

const licenseHeader = "# ---------------------------------------------------------------------------\n" +
	"# Licensed to the Apache Software Foundation (ASF) under one or more\n" +
	"# contributor license agreements.  See the NOTICE file distributed with\n" +
	"# this work for additional information regarding copyright ownership.\n" +
	"# The ASF licenses this file to You under the Apache License, Version 2.0\n" +
	"# (the \"License\"); you may not use this file except in compliance with\n" +
	"# the License.  You may obtain a copy of the License at\n" +
	"#\n" +
	"#      http://www.apache.org/licenses/LICENSE-2.0\n" +
	"#\n" +
	"# Unless required by applicable law or agreed to in writing, software\n" +
	"# distributed under the License is distributed on an \"AS IS\" BASIS,\n" +
	"# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n" +
	"# See the License for the specific language governing permissions and\n" +
	"# limitations under the License.\n" +
	"# ---------------------------------------------------------------------------\n"

// traitMetaDataGen produces YAML documentation about trait descriptions.
type traitMetaDataGen struct {
	generator.DefaultGen
	arguments *args.GeneratorArgs
	Root      *traitMetaDataRoot
}

type traitMetaDataRoot struct {
	Traits []traitMetaData `yaml:"traits"`
}

type traitMetaData struct {
	Name        string                  `yaml:"name"`
	Platform    bool                    `yaml:"platform"`
	Profiles    []string                `yaml:"profiles"`
	Description string                  `yaml:"description"`
	Properties  []traitPropertyMetaData `yaml:"properties"`
}

type traitPropertyMetaData struct {
	Name        string `yaml:"name"`
	TypeName    string `yaml:"type"`
	Description string `yaml:"description"`
}

// traitMetaDataGen implements Generator interface.
var _ generator.Generator = &traitMetaDataGen{}

// NewtraitMetaDataGen --.
func NewtraitMetaDataGen(arguments *args.GeneratorArgs) generator.Generator {
	return &traitMetaDataGen{
		DefaultGen: generator.DefaultGen{},
		arguments:  arguments,
		Root:       &traitMetaDataRoot{},
	}
}

func (g *traitMetaDataGen) Filename() string {
	return "zz_desc_generated.go"
}

func (g *traitMetaDataGen) Filter(context *generator.Context, t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, tagTrait) {
			return true
		}
	}
	return false
}

func (g *traitMetaDataGen) GenerateType(context *generator.Context, t *types.Type, out io.Writer) error {
	traitID := g.getTraitID(t)
	td := &traitMetaData{}
	g.buildDescription(t, traitID, td)
	g.buildFields(t, td)
	g.Root.Traits = append(g.Root.Traits, *td)
	return nil
}

func (g *traitMetaDataGen) Finalize(c *generator.Context, w io.Writer) error {
	customArgs, ok := g.arguments.CustomArgs.(*CustomArgs)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", g.arguments.CustomArgs)
	}
	deployDir := customArgs.ResourceDir
	filename := path.Join(deployDir, traitFile)

	// reorder the traits metadata so that it always gets the identical result
	sort.Slice(g.Root.Traits, func(i, j int) bool {
		return g.Root.Traits[i].Name < g.Root.Traits[j].Name
	})

	return util.WithFile(filename, os.O_RDWR|os.O_CREATE, 0o777, func(file *os.File) error {
		if err := file.Truncate(0); err != nil {
			return err
		}

		fmt.Fprintf(file, "%s", string(licenseHeader))
		data, err := yaml.Marshal(g.Root)
		if err != nil {
			fmt.Fprintf(file, "error: %v", err)
		}
		fmt.Fprintf(file, "%s", string(data))

		return nil
	})
}

func (g *traitMetaDataGen) getTraitID(t *types.Type) string {
	for _, s := range t.CommentLines {
		if strings.Contains(s, tagTrait) {
			matches := tagTraitID.FindStringSubmatch(s)
			if len(matches) < 2 {
				panic(fmt.Sprintf("unable to extract trait ID from tag line `%s`", s))
			}
			return matches[1]
		}
	}
	panic(fmt.Sprintf("trait ID not found in type %s", t.Name.Name))
}

func (g *traitMetaDataGen) buildDescription(t *types.Type, traitID string, td *traitMetaData) {
	desc := []string(nil)
	desc = append(desc, filterOutTagsAndComments(t.CommentLines)...)
	td.Name = traitID
	td.Description = ""
	for _, line := range desc {
		text := strings.Trim(line, " ")
		if len(text) == 0 {
			continue
		}
		if len(td.Description) > 0 {
			td.Description += " "
		}
		td.Description += text
	}
	td.Profiles = determineProfiles(traitID)
	td.Platform = isPlatformTrait(traitID)
}

func (g *traitMetaDataGen) buildFields(t *types.Type, td *traitMetaData) {
	if len(t.Members) > 1 {
		res := []string(nil)
		g.buildMembers(t, &res, td)
	}
}

func (g *traitMetaDataGen) buildMembers(t *types.Type, content *[]string, td *traitMetaData) {
	for _, m := range t.Members {
		res := append([]string(nil), *content...)
		prop := reflect.StructTag(m.Tags).Get("property")
		if prop != "" {
			if strings.Contains(prop, "squash") {
				g.buildMembers(m.Type, &res, td)
			} else {
				pd := traitPropertyMetaData{}
				pd.Name = prop
				pd.TypeName = strings.TrimPrefix(m.Type.Name.Name, "*")

				res = append(res, filterOutTagsAndComments(m.CommentLines)...)
				pd.Description = strings.Join(res, " ")
				td.Properties = append(td.Properties, pd)
			}
		}
	}
}
