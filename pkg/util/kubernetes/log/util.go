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
	"bytes"
	"context"
	"fmt"
	"io"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Print prints integrations logs to the stdout.
func Print(ctx context.Context, cmd *cobra.Command, client kubernetes.Interface, integration *v1.Integration, tailLines *int64, out io.Writer) error {
	return PrintUsingSelector(ctx, cmd, client, integration.Namespace, integration.Name, v1.IntegrationLabel+"="+integration.Name, tailLines, out)
}

// PrintUsingSelector prints pod logs using a selector.
func PrintUsingSelector(ctx context.Context, cmd *cobra.Command, client kubernetes.Interface, namespace, defaultContainerName, selector string, tailLines *int64, out io.Writer) error {
	scraper := NewSelectorScraper(client, namespace, defaultContainerName, selector, tailLines)
	reader := scraper.Start(ctx)

	if _, err := io.Copy(out, io.NopCloser(reader)); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
	}

	return nil
}

// DumpLog extract the full log from a Pod. Recommended when the quantity of log expected is minimum.
func DumpLog(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod, podLogOpts corev1.PodLogOptions) (string, error) {
	req := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()

	return str, nil
}
