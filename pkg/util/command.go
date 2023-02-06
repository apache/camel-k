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

package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// RunAndLog starts the provided command, scans its standard and error outputs line by line,
// to feed the provided handlers, and waits until the scans complete and the command returns.
func RunAndLog(ctx context.Context, cmd *exec.Cmd, stdOutF func(string) string, stdErrF func(string) string) error {
	scanOutMsg := ""
	scanErrMsg := ""
	stdOutF(fmt.Sprintf("Executed command: %s", cmd.String()))

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	// if the command is in error, we try to figure it out why also by parsing the log
	if err != nil {
		scanOutMsg = scan(stdOut, stdOutF)
		scanErrMsg = scan(stdErr, stdErrF)

		return errors.Wrapf(err, formatErr(scanOutMsg, scanErrMsg))
	}
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		scanOutMsg = scan(stdOut, stdOutF)
		return nil
	})
	g.Go(func() error {
		scanErrMsg = scan(stdErr, stdErrF)
		return nil
	})
	if err = g.Wait(); err != nil {
		return errors.Wrapf(err, formatErr(scanOutMsg, scanErrMsg))
	}
	if err = cmd.Wait(); err != nil {
		return errors.Wrapf(err, formatErr(scanOutMsg, scanErrMsg))
	}

	return nil
}

func scan(stdOut io.ReadCloser, stdOutF func(string) string) string {
	scanError := ""
	scanner := bufio.NewScanner(stdOut)
	for scanner.Scan() {
		errMsg := stdOutF(scanner.Text())
		if errMsg != "" && scanError == "" {
			scanError = errMsg
		}
	}

	return scanError
}

func formatErr(stdout, stderr string) string {
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return fmt.Sprintf("stdout: %s, stderr: %s", stdout, stderr)
}
