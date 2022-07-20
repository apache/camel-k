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

package log

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/spf13/cobra"

	"k8s.io/client-go/kubernetes"
)

// Print prints integrations logs to the stdout.
func Print(ctx context.Context, cmd *cobra.Command, client kubernetes.Interface, integration *v1.Integration,
	out io.Writer) error {
	return PrintUsingSelector(ctx, cmd, client, integration.Namespace, integration.Name,
		v1.IntegrationLabel+"="+integration.Name, out)
}

// PrintUsingSelector prints pod logs using a selector.
func PrintUsingSelector(ctx context.Context, cmd *cobra.Command, client kubernetes.Interface,
	namespace, defaultContainerName, selector string, out io.Writer) error {
	scraper := NewSelectorScraper(client, namespace, defaultContainerName, selector)
	reader := scraper.Start(ctx)

	if _, err := io.Copy(out, ioutil.NopCloser(reader)); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}

	return nil
}
