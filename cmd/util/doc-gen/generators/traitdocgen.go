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
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util"
)

const (
	tagTrait      = "+camel-k:trait"
	tagDeprecated = "+camel-k:deprecated"
	tagLint       = "nolint"

	adocCommonMarkerStart = "// Start of autogenerated code - DO NOT EDIT!"
	adocCommonMarkerEnd   = "// End of autogenerated code - DO NOT EDIT!"

	adocBadgesMarkerStart = adocCommonMarkerStart + " (badges)"
	adocBadgesMarkerEnd   = adocCommonMarkerEnd + " (badges)"

	adocDescriptionMarkerStart = adocCommonMarkerStart + " (description)"
	adocDescriptionMarkerEnd   = adocCommonMarkerEnd + " (description)"

	adocConfigurationMarkerStart = adocCommonMarkerStart + " (configuration)"
	adocConfigurationMarkerEnd   = adocCommonMarkerEnd + " (configuration)"

	adocNavMarkerStart = adocCommonMarkerStart + " (trait-nav)"
	adocNavMarkerEnd   = adocCommonMarkerEnd + " (trait-nav)"
)

var tagTraitRegex = regexp.MustCompile(fmt.Sprintf("%s=([a-z0-9-]+)", regexp.QuoteMeta(tagTrait)))
var tagDeprecatedRegex = regexp.MustCompile(fmt.Sprintf("%s=(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)", regexp.QuoteMeta(tagDeprecated)))

// traitDocGen produces documentation about traits.
type traitDocGen struct {
	generator.DefaultGen
	arguments           *args.GeneratorArgs
	generatedTraitFiles []string
}

// traitDocGen implements Generator interface.
var _ generator.Generator = &traitDocGen{}

func NewTraitDocGen(arguments *args.GeneratorArgs) generator.Generator {
	return &traitDocGen{
		DefaultGen: generator.DefaultGen{},
		arguments:  arguments,
	}
}

func (g *traitDocGen) Filename() string {
	return "zz_generated_doc.go"
}

func (g *traitDocGen) Filter(context *generator.Context, t *types.Type) bool {
	for _, c := range t.CommentLines {
		if strings.Contains(c, tagTrait) {
			return true
		}
	}
	return false
}

func (g *traitDocGen) GenerateType(context *generator.Context, t *types.Type, out io.Writer) error {
	customArgs, ok := g.arguments.CustomArgs.(*CustomArgs)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", g.arguments.CustomArgs)
	}
	docDir := customArgs.DocDir
	traitPath := customArgs.TraitPath
	traitID := getTraitID(t)
	traitFile := traitID + ".adoc"
	filename := path.Join(docDir, traitPath, traitFile)

	g.generatedTraitFiles = append(g.generatedTraitFiles, traitFile)

	return util.WithFileContent(filename, func(file *os.File, data []byte) error {
		content := strings.Split(string(data), "\n")

		writeTitle(t, traitID, &content)
		writeBadges(t, traitID, &content)
		writeDescription(t, traitID, &content)
		writeFields(t, traitID, &content)

		return writeFile(file, content)
	})
}

func (g *traitDocGen) Finalize(c *generator.Context, w io.Writer) error {
	return g.FinalizeNav(c)
}

func (g *traitDocGen) FinalizeNav(*generator.Context) error {
	customArgs, ok := g.arguments.CustomArgs.(*CustomArgs)
	if !ok {
		return fmt.Errorf("type assertion failed: %v", g.arguments.CustomArgs)
	}
	docDir := customArgs.DocDir
	navPath := customArgs.NavPath
	filename := path.Join(docDir, navPath)

	return util.WithFileContent(filename, func(file *os.File, data []byte) error {
		content := strings.Split(string(data), "\n")

		pre, post := split(content, adocNavMarkerStart, adocNavMarkerEnd)

		content = append([]string(nil), pre...)
		content = append(content, adocNavMarkerStart)
		sort.Strings(g.generatedTraitFiles)
		for _, t := range g.generatedTraitFiles {
			name := traitNameFromFile(t)
			content = append(content, "** xref:traits:"+t+"["+name+"]")
		}
		content = append(content, adocNavMarkerEnd)
		content = append(content, post...)

		return writeFile(file, content)
	})
}

func traitNameFromFile(file string) string {
	name := strings.TrimSuffix(file, ".adoc")
	name = strings.ReplaceAll(name, "trait", "")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Trim(name, " ")
	name = cases.Title(language.English).String(name)
	return name
}

func writeTitle(t *types.Type, traitID string, content *[]string) {
	res := append([]string(nil), *content...)
	// Check if we already have a title
	for _, s := range res {
		if strings.HasPrefix(s, "= ") {
			return
		}
	}
	res = append([]string{"= " + cases.Title(language.English).String(strings.ReplaceAll(traitID, "-", " ")) + " Trait"}, res...)
	*content = res
}

// Write badges
// https://shields.io/badges/static-badge
func writeBadges(t *types.Type, traitID string, content *[]string) {
	pre, post := split(*content, adocBadgesMarkerStart, adocBadgesMarkerEnd)
	// When there are no badges in the generated output already
	// assume that we want to have them just after the title
	if len(post) == 0 {
		pre = (*content)[:2]
		post = (*content)[2:]
	}
	res := append([]string(nil), pre...)
	res = append(res, adocBadgesMarkerStart)
	if ver := getDeprecatedVersion(t); ver != "" {
		res = append(res, "image:https://img.shields.io/badge/"+ver+"-white?label=Deprecated&labelColor=C40C0C&color=gray[Deprecated Badge]")
	}
	res = append(res, adocBadgesMarkerEnd)
	res = append(res, post...)
	*content = res
}

func writeDescription(t *types.Type, traitID string, content *[]string) {
	pre, post := split(*content, adocDescriptionMarkerStart, adocDescriptionMarkerEnd)
	res := append([]string(nil), pre...)
	res = append(res, adocDescriptionMarkerStart)
	res = append(res, filterOutTagsAndComments(t.CommentLines)...)
	profiles := strings.Join(determineProfiles(traitID), ", ")
	res = append(res, "", fmt.Sprintf("This trait is available in the following profiles: **%s**.", profiles))
	if isPlatformTrait(traitID) {
		res = append(res, "", fmt.Sprintf("NOTE: The %s trait is a *platform trait* and cannot be disabled by the user.", traitID))
	}
	res = append(res, "", adocDescriptionMarkerEnd)
	res = append(res, post...)
	*content = res
}

func writeFields(t *types.Type, traitID string, content *[]string) {
	pre, post := split(*content, adocConfigurationMarkerStart, adocConfigurationMarkerEnd)
	res := append([]string(nil), pre...)
	res = append(res, adocConfigurationMarkerStart, "== Configuration", "")
	res = append(res, "Trait properties can be specified when running any integration with the CLI:")
	res = append(res, "[source,console]")
	res = append(res, "----")
	if len(t.Members) > 1 {
		res = append(res, fmt.Sprintf("$ kamel run --trait %s.[key]=[value] --trait %s.[key2]=[value2] integration.groovy", traitID, traitID))
	} else {
		res = append(res, fmt.Sprintf("$ kamel run --trait %s.[key]=[value] integration.groovy", traitID))
	}
	res = append(res, "----")
	res = append(res, "The following configuration options are available:", "")
	res = append(res, "[cols=\"2m,1m,5a\"]", "|===")
	res = append(res, "|Property | Type | Description", "")
	writeMembers(t, traitID, &res)
	res = append(res, "|===", "", adocConfigurationMarkerEnd)
	res = append(res, post...)
	*content = res
}

func writeMembers(t *types.Type, traitID string, content *[]string) {
	res := append([]string(nil), *content...)
	for _, m := range t.Members {
		prop := reflect.StructTag(m.Tags).Get("property")
		if prop == "" {
			continue
		}

		if strings.Contains(prop, "squash") {
			writeMembers(m.Type, traitID, &res)
		} else {
			res = append(res, "| "+traitID+"."+prop)
			res = append(res, "| "+strings.TrimPrefix(m.Type.Name.Name, "*"))
			first := true
			for _, l := range filterOutTagsAndComments(m.CommentLines) {
				escapedComment := escapeASCIIDoc(l)
				if first {
					res = append(res, "| "+escapedComment)
					first = false
				} else {
					res = append(res, escapedComment)
				}
			}
			res = append(res, "")
		}
	}
	*content = res
}

func getTraitID(t *types.Type) string {
	for _, s := range t.CommentLines {
		if strings.Contains(s, tagTrait) {
			matches := tagTraitRegex.FindStringSubmatch(s)
			if len(matches) < 2 {
				panic(fmt.Sprintf("unable to extract trait ID from tag line `%s`", s))
			}
			return matches[1]
		}
	}
	panic(fmt.Sprintf("trait ID not found in type %s", t.Name.Name))
}

func getDeprecatedVersion(t *types.Type) string {
	for _, s := range t.CommentLines {
		if strings.Contains(s, tagDeprecated) {
			matches := tagDeprecatedRegex.FindStringSubmatch(s)
			if len(matches) == 4 {
				return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
			}
		}
	}
	return ""
}

func filterOutTagsAndComments(comments []string) []string {
	res := make([]string, 0, len(comments))
	for _, l := range comments {
		if !strings.HasPrefix(strings.TrimLeft(l, " \t"), "+") &&
			!strings.HasPrefix(strings.TrimLeft(l, " \t"), "TODO:") &&
			!strings.HasPrefix(strings.TrimLeft(l, " \t"), tagLint) {
			res = append(res, l)
		}
	}
	return res
}

// escapeAsciiDoc is in charge to escape those chars used for formatting purposes.
func escapeASCIIDoc(text string) string {
	return strings.ReplaceAll(text, "|", "\\|")
}

func split(doc []string, startMarker, endMarker string) ([]string, []string) {
	if len(doc) == 0 {
		return nil, nil
	}
	idx := len(doc)
	for i, s := range doc {
		if s == startMarker {
			idx = i
			break
		}
	}
	idy := len(doc)
	for j, s := range doc {
		if j > idx && s == endMarker {
			idy = j
			break
		}
	}
	pre := doc[0:idx]
	post := []string{}
	if idy < len(doc) {
		post = doc[idy+1:]
	}
	return pre, post
}

func writeFile(file *os.File, content []string) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	max := 0
	for i, line := range content {
		if line != "" {
			max = i
		}
	}
	for i, line := range content {
		if i <= max {
			if _, err := file.WriteString(line + "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func isPlatformTrait(traitID string) bool {
	catalog := trait.NewCatalog(nil)
	t := catalog.GetTrait(traitID)
	return t.IsPlatformTrait()
}

func determineProfiles(traitID string) []string {
	var profiles []string
	catalog := trait.NewCatalog(nil)
	for _, p := range v1.AllTraitProfiles {
		traits := catalog.TraitsForProfile(p)
		for _, t := range traits {
			if string(t.ID()) == traitID {
				profiles = append(profiles, string(p))
			}
		}
	}
	return profiles
}
