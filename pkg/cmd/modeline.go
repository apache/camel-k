package cmd

import (
	"context"
	"fmt"
	"path"

	"github.com/apache/camel-k/pkg/util/modeline"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"path/filepath"
)

const (
	runCmdName        = "run"
	runCmdSourcesArgs = "source"
)

var (
	nonRunOptions = map[string]bool{
		"language": true, // language is a marker modeline option for other tools
	}

	// file options must be considered relative to the source files they belong to
	fileOptions = map[string]bool{
		"source":        true,
		"resource":      true,
		"config":        true,
		"open-api":      true,
		"property-file": true,
	}
)

func NewKamelWithModelineCommand(ctx context.Context, osArgs []string) (*cobra.Command, []string, error) {
	processed := make(map[string]bool)
	originalFlags := osArgs[1:]
	rootCmd, flags, err := createKamelWithModelineCommand(ctx, append([]string(nil), originalFlags...), processed)
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

func createKamelWithModelineCommand(ctx context.Context, args []string, processedFiles map[string]bool) (*cobra.Command, []string, error) {
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

	sources, err := fg.GetStringArray(runCmdSourcesArgs)
	if err != nil {
		return nil, nil, err
	}

	var files = append([]string(nil), fg.Args()...)
	files = append(files, sources...)

	opts := make([]modeline.Option, 0)
	for _, f := range files {
		if processedFiles[f] {
			continue
		}
		baseDir := filepath.Dir(f)
		content, err := loadData(f, false)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot read file %s", f)
		}
		ops, err := modeline.Parse(f, content)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot process file %s", f)
		}
		for i, o := range ops {
			if fileOptions[o.Name] && !isRemoteHTTPFile(f) {
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
	// filter out in place non-run options
	nOpts := 0
	for _, o := range opts {
		if !nonRunOptions[o.Name] {
			opts[nOpts] = o
			nOpts++
		}
	}
	opts = opts[:nOpts]

	// No new options, returning a new command with computed args
	if len(opts) == 0 {
		// Recreating the command as it's dirty
		rootCmd, err = NewKamelCommand(ctx)
		if err != nil {
			return nil, nil, err
		}
		rootCmd.SetArgs(args)
		return rootCmd, args, nil
	}

	// New options added, recomputing
	for _, f := range files {
		processedFiles[f] = true
	}
	for _, o := range opts {
		prefix := "-"
		if len(o.Name) > 1 {
			prefix = "--"
		}
		args = append(args, fmt.Sprintf("%s%s", prefix, o.Name))
		args = append(args, o.Value)
	}

	return createKamelWithModelineCommand(ctx, args, processedFiles)
}
