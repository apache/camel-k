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

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/resources"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func newTraitHelpCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *traitHelpCommandOptions) {
	options := traitHelpCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "trait",
		Short:   "Trait help information",
		Long:    `Displays help information for traits in a specified output format.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			return options.run(cmd, args)
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	cmd.Flags().Bool("all", false, "Include all traits")
	cmd.Flags().StringP("output", "o", "", "Output format. One of json, yaml")

	return &cmd, &options
}

type traitHelpCommandOptions struct {
	*RootCmdOptions
	IncludeAll   bool   `mapstructure:"all"`
	OutputFormat string `mapstructure:"output"`
}

type traitDescription struct {
	Name        trait.ID                   `json:"name" yaml:"name"`
	Platform    bool                       `json:"platform" yaml:"platform"`
	Profiles    []string                   `json:"profiles" yaml:"profiles"`
	Properties  []traitPropertyDescription `json:"properties" yaml:"properties"`
	Description string                     `json:"description" yaml:"description"`
}

type traitPropertyDescription struct {
	Name         string      `json:"name" yaml:"name"`
	TypeName     string      `json:"type" yaml:"type"`
	DefaultValue interface{} `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"`
	Description  string      `json:"description" yaml:"description"`
}

type traitMetaData struct {
	Traits []traitDescription `yaml:"traits"`
}

func (command *traitHelpCommandOptions) validate(args []string) error {
	if command.IncludeAll && len(args) > 0 {
		return errors.New("invalid combination: both all flag and a named trait is set")
	}
	if !command.IncludeAll && len(args) == 0 {
		return errors.New("invalid combination: neither all flag nor a named trait is set")
	}
	return nil
}

func (command *traitHelpCommandOptions) run(cmd *cobra.Command, args []string) error {
	var traitDescriptions []*traitDescription
	catalog := trait.NewCatalog(nil)

	traitMetaData := &traitMetaData{}
	err := yaml.Unmarshal(resources.Resource("/traits.yaml"), traitMetaData)
	if err != nil {
		return err
	}

	for _, tp := range v1.AllTraitProfiles {
		traits := catalog.TraitsForProfile(tp)
		for _, t := range traits {
			if len(args) == 1 && trait.ID(args[0]) != t.ID() {
				continue
			}

			td := findTraitDescription(t.ID(), traitDescriptions)
			if td == nil {
				td = &traitDescription{
					Name:     t.ID(),
					Platform: t.IsPlatformTrait(),
					Profiles: make([]string, 0),
				}

				var targetTrait *traitDescription
				for _, item := range traitMetaData.Traits {
					item := item
					if item.Name == t.ID() {
						targetTrait = &item
						td.Description = item.Description
						break
					}
				}
				computeTraitProperties(structs.Fields(t), &td.Properties, targetTrait)
				traitDescriptions = append(traitDescriptions, td)
			}
			td.addProfile(string(tp))
		}
	}

	if len(args) == 1 && len(traitDescriptions) == 0 {
		return fmt.Errorf("no trait named '%s' exists", args[0])
	}

	switch strings.ToUpper(command.OutputFormat) {
	case "JSON":
		res, err := json.Marshal(traitDescriptions)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(res))
	case "YAML":
		res, err := yaml.Marshal(traitDescriptions)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(res))
	default:
		res, err := outputTraits(traitDescriptions)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), res)
	}

	return nil
}

func (td *traitDescription) addProfile(tp string) {
	for _, p := range td.Profiles {
		if p == tp {
			return
		}
	}
	td.Profiles = append(td.Profiles, tp)
}

func findTraitDescription(id trait.ID, traitDescriptions []*traitDescription) *traitDescription {
	for _, td := range traitDescriptions {
		if td.Name == id {
			return td
		}
	}
	return nil
}

func computeTraitProperties(fields []*structs.Field, properties *[]traitPropertyDescription, targetTrait *traitDescription) {
	for _, f := range fields {
		if f.IsEmbedded() && f.IsExported() && f.Kind() == reflect.Struct {
			computeTraitProperties(f.Fields(), properties, targetTrait)
		}

		if !f.IsExported() || f.IsEmbedded() {
			continue
		}

		property := f.Tag("property")
		if property == "" {
			continue
		}

		tp := traitPropertyDescription{}
		tp.Name = property

		switch f.Kind() {
		case reflect.Ptr:
			tp.TypeName = reflect.TypeOf(f.Value()).Elem().String()
		case reflect.Slice:
			tp.TypeName = fmt.Sprintf("slice:%s", reflect.TypeOf(f.Value()).Elem().String())
		default:
			tp.TypeName = f.Kind().String()
		}

		if f.IsZero() {
			if tp.TypeName == "bool" {
				tp.DefaultValue = false
			} else {
				tp.DefaultValue = nil
			}
		} else {
			tp.DefaultValue = f.Value()
		}

		// apply the description from metadata
		if targetTrait != nil {
			for _, item := range targetTrait.Properties {
				if item.Name == tp.Name {
					tp.Description = item.Description
				}
			}
		}

		*properties = append(*properties, tp)
	}
}

func outputTraits(descriptions []*traitDescription) (string, error) {
	return indentedwriter.IndentedString(func(out io.Writer) error {
		w := indentedwriter.NewWriter(out)

		for _, td := range descriptions {
			w.Writef(0, "Name:\t%s\n", td.Name)
			w.Writef(0, "Profiles:\t%s\n", strings.Join(td.Profiles, ","))
			w.Writef(0, "Platform:\t%t\n", td.Platform)
			w.Writef(0, "Properties:\n")
			for _, p := range td.Properties {
				w.Writef(1, "%s:\n", p.Name)
				w.Writef(2, "Type:\t%s\n", p.TypeName)
				if p.DefaultValue != nil {
					w.Writef(2, "Default Value:\t%v\n", p.DefaultValue)
				}
			}
			w.Writelnf(0, "")
		}

		return nil
	})
}
