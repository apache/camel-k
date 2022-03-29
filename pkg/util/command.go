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
	"os/exec"

	"golang.org/x/sync/errgroup"
)

// RunAndLog starts the provided command, scans its standard and error outputs line by line,
// to feed the provided handlers, and waits until the scans complete and the command returns.
func RunAndLog(ctx context.Context, cmd *exec.Cmd, stdOutF func(string), stdErrF func(string)) (err error) {
	stdOutF(fmt.Sprintf("Executed command: %s", cmd.String()))

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	err = cmd.Start()
	if err != nil {
		scanOut := bufio.NewScanner(stdOut)
		for scanOut.Scan() {
			stdOutF(scanOut.Text())
		}
		scanErr := bufio.NewScanner(stdErr)
		for scanErr.Scan() {
			stdOutF(scanErr.Text())
		}
		return
	}
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		scanner := bufio.NewScanner(stdOut)
		for scanner.Scan() {
			stdOutF(scanner.Text())
		}
		return nil
	})
	g.Go(func() error {
		scanner := bufio.NewScanner(stdErr)
		for scanner.Scan() {
			stdErrF(scanner.Text())
		}
		return nil
	})
	err = g.Wait()
	if err != nil {
		return
	}
	err = cmd.Wait()
	if err != nil {
		return
	}
	return
}
