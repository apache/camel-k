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
	"bufio"
	"context"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

// PodScraper scrapes logs of a specific pod
type PodScraper struct {
	namespace string
	name      string
}

// NewPodScraper creates a new pod scraper
func NewPodScraper(namespace string, name string) *PodScraper {
	return &PodScraper{
		namespace: namespace,
		name:      name,
	}
}

// Start returns a reader that streams the pod logs
func (s *PodScraper) Start(ctx context.Context) *bufio.Reader {
	pipeIn, pipeOut := io.Pipe()
	bufPipeIn := bufio.NewReader(pipeIn)
	bufPipeOut := bufio.NewWriter(pipeOut)
	closeFun := func() error {
		bufPipeOut.Flush()
		return pipeOut.Close()
	}
	go s.doScrape(ctx, bufPipeOut, closeFun)
	return bufPipeIn
}

func (s *PodScraper) doScrape(ctx context.Context, out *bufio.Writer, clientCloser func() error) {
	err := s.waitForPodRunning(ctx, s.namespace, s.name)
	if err != nil {
		s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
		return
	}

	byteReader, err := k8sclient.GetKubeClient().CoreV1().Pods(s.namespace).GetLogs(s.name, &v1.PodLogOptions{Follow: true}).Context(ctx).Stream()
	if err != nil {
		s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
		return
	}

	reader := bufio.NewReader(byteReader)
	err = nil
	for err == nil {
		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		_, err = out.WriteString(str)
		if err != nil {
			break
		}
		out.Flush()
	}
	if err == io.EOF {
		return
	}

	s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
}

func (s *PodScraper) handleAndRestart(ctx context.Context, err error, wait time.Duration, out *bufio.Writer, clientCloser func() error) {
	if err != nil {
		logrus.Warn(errors.Wrap(err, "error caught during log scraping for pod "+s.name))
	}

	if ctx.Err() != nil {
		logrus.Info("Pod ", s.name, " will no longer be monitored")
		clientCloser()
		return
	}

	logrus.Info("Retrying to scrape pod ", s.name, " logs in ", wait.Seconds(), " seconds...")
	select {
	case <-time.After(wait):
		break
	case <-ctx.Done():
		clientCloser()
		return
	}

	s.doScrape(ctx, out, clientCloser)
}

// Waits for a given pod to reach the running state
func (s *PodScraper) waitForPodRunning(ctx context.Context, namespace string, name string) error {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	resourceClient, _, err := k8sclient.GetResourceClient(pod.APIVersion, pod.Kind, pod.Namespace)
	if err != nil {
		return err
	}
	watcher, err := resourceClient.Watch(metav1.ListOptions{
		FieldSelector: "metadata.name=" + pod.Name,
	})
	if err != nil {
		return err
	}
	events := watcher.ResultChan()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e, ok := <-events:
			if !ok {
				return errors.New("event channel closed")
			}

			if e.Object != nil {
				if runtimeUnstructured, ok := e.Object.(runtime.Unstructured); ok {
					unstr := unstructured.Unstructured{
						Object: runtimeUnstructured.UnstructuredContent(),
					}
					pcopy := pod.DeepCopy()
					err := k8sutil.UnstructuredIntoRuntimeObject(&unstr, pcopy)
					if err != nil {
						return err
					}

					if pcopy.Status.Phase == v1.PodRunning {
						return nil
					}
				}
			} else if e.Type == watch.Deleted || e.Type == watch.Error {
				return errors.New("unable to watch pod " + s.name)
			}
		case <-time.After(30 * time.Second):
			return errors.New("no state change after 30 seconds for pod " + s.name)
		}
	}

	return nil
}
