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
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/apache/camel-k/pkg/util/modeline"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	runCmdName        = "run"
	runCmdSourcesArgs = "source"
)

var (
	nonRunOptions = map[string]bool{
		"language": true, // language is a marker modeline option for other tools
	}
	disallowedOptions = map[string]bool{
		"dev":  true,
		"wait": true,
		"logs": true,
		"sync": true,
	}

	// file options must be considered relative to the source files they belong to
	fileOptions = map[string]bool{
		"resource":      true,
		"config":        true,
		"open-api":      true,
		"property-file": true,
	}
)

// NewKamelWithModelineCommand ---
func NewKamelWithModelineCommand(ctx context.Context, osArgs []string) (*cobra.Command, []string, error) {
	originalFlags := osArgs[1:]
	rootCmd, flags, err := createKamelWithModelineCommand(ctx, originalFlags)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return rootCmd, flags, err
	}
	if len(originalFlags) != len(flags) {
		// Give a feedback about the actual command that is run
		fmt.Fprintln(rootCmd.OutOrStdout(), "Modeline options have been loaded from source files")
		fmt.Fprint(rootCmd.OutOrStdout(), "Full command: kamel ")
		for _, a := range flags {
			fmt.Fprintf(rootCmd.OutOrStdout(), "%s ", a)
		}
		fmt.Fprintln(rootCmd.OutOrStdout())
	}
	return rootCmd, flags, nil
}

func createKamelWithModelineCommand(ctx context.Context, args []string) (*cobra.Command, []string, error) {
	rootCmd, err := NewKamelCommand(ctx)
	if err != nil {
		return nil, nil, err
	}

	target, flags, err := rootCmd.Find(args)
	if err != nil {
		return nil, nil, err
	}

	if target.Name() != runCmdName {
		return rootCmd, args, nil
	}

	err = target.ParseFlags(flags)
	if err == pflag.ErrHelp {
		return rootCmd, args, nil
	} else if err != nil {
		return nil, nil, err
	}

	fg := target.Flags()

	additionalSources, err := fg.GetStringArray(runCmdSourcesArgs)
	if err != nil {
		return nil, nil, err
	}

	files := make([]string, 0, len(fg.Args())+len(additionalSources))
	files = append(files, fg.Args()...)
	files = append(files, additionalSources...)

	opts, err := extractModelineOptions(ctx, files)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot read sources")
	}

	// filter out in place non-run options
	nOpts := 0
	for _, o := range opts {
		if !nonRunOptions[o.Name] {
			opts[nOpts] = o
			nOpts++
		}
	}

	opts = opts[:nOpts]

	for _, o := range opts {
		prefix := "-"
		if len(o.Name) > 1 {
			prefix = "--"
		}
		// Using the k=v syntax to avoid issues with booleans
		if len(o.Value) > 0 {
			args = append(args, fmt.Sprintf("%s%s=%s", prefix, o.Name, o.Value))
		} else {
			args = append(args, fmt.Sprintf("%s%s", prefix, o.Name))
		}
	}

	// Recreating the command as it's dirty
	rootCmd, err = NewKamelCommand(ctx)
	if err != nil {
		return nil, nil, err
	}
	rootCmd.SetArgs(args)

	return rootCmd, args, nil
}

func extractModelineOptions(ctx context.Context, sources []string) ([]modeline.Option, error) {
	opts := make([]modeline.Option, 0)

	resolvedSources, err := ResolveSources(ctx, sources, false)
	if err != nil {
		return opts, errors.Wrap(err, "cannot read sources")
	}

	for _, resolvedSource := range resolvedSources {
		ops, err := modeline.Parse(resolvedSource.Location, resolvedSource.Content)
		if err != nil {
			return opts, errors.Wrapf(err, "cannot process file %s", resolvedSource.Location)
		}
		for i, o := range ops {
			if disallowedOptions[o.Name] {
				return opts, fmt.Errorf("option %q is disallowed in modeline", o.Name)
			}

			if fileOptions[o.Name] && resolvedSource.Local {
				baseDir := filepath.Dir(resolvedSource.Origin)
				refPath := o.Value
				if !filepath.IsAbs(refPath) {
					full := path.Join(baseDir, refPath)
					o.Value = full
					ops[i] = o
				}
			}
		}
		opts = append(opts, ops...)
	}

	return opts, nil
}
