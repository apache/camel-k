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
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

var commonUserContainerNames = map[string]bool{
	// Convention used in Knative and Istio
	"user-container": true,
}

// PodScraper scrapes logs of a specific pod
type PodScraper struct {
	namespace string
	name      string
	client    kubernetes.Interface
}

// NewPodScraper creates a new pod scraper
func NewPodScraper(c kubernetes.Interface, namespace string, name string) *PodScraper {
	return &PodScraper{
		namespace: namespace,
		name:      name,
		client:    c,
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
	containerName, err := s.waitForPodRunning(ctx, s.namespace, s.name)
	if err != nil {
		s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
		return
	}
	logOptions := v1.PodLogOptions{
		Follow:    true,
		Container: containerName,
	}
	byteReader, err := s.client.CoreV1().Pods(s.namespace).GetLogs(s.name, &logOptions).Context(ctx).Stream()
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
		logrus.Debug("Pod ", s.name, " will no longer be monitored")
		if err := clientCloser(); err != nil {
			logrus.Warn("Unable to close the client", err)
		}
		return
	}

	logrus.Debug("Retrying to scrape pod ", s.name, " logs in ", wait.Seconds(), " seconds...")
	select {
	case <-time.After(wait):
		break
	case <-ctx.Done():
		if err := clientCloser(); err != nil {
			logrus.Warn("Unable to close the client", err)
		}
		return
	}

	s.doScrape(ctx, out, clientCloser)
}

// waitForPodRunning waits for a given pod to reach the running state.
// It may return the internal container to watch if present
func (s *PodScraper) waitForPodRunning(ctx context.Context, namespace string, name string) (string, error) {
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
	podClient := s.client.CoreV1().Pods(pod.Namespace)
	watcher, err := podClient.Watch(metav1.ListOptions{
		FieldSelector: "metadata.name=" + pod.Name,
	})
	if err != nil {
		return "", err
	}
	events := watcher.ResultChan()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case e, ok := <-events:
			if !ok {
				return "", errors.New("event channel closed")
			}

			if e.Object != nil {
				var recvPod *v1.Pod
				if runtimeUnstructured, ok := e.Object.(runtime.Unstructured); ok {
					unstr := unstructured.Unstructured{
						Object: runtimeUnstructured.UnstructuredContent(),
					}
					jsondata, err := unstr.MarshalJSON()
					if err != nil {
						return "", err
					}
					recvPod := pod.DeepCopy()
					if err := json.Unmarshal(jsondata, recvPod); err != nil {
						return "", err
					}

				} else if gotPod, ok := e.Object.(*v1.Pod); ok {
					recvPod = gotPod
				}

				if recvPod != nil && recvPod.Status.Phase == v1.PodRunning {
					return s.chooseContainer(recvPod), nil
				}
			} else if e.Type == watch.Deleted || e.Type == watch.Error {
				return "", errors.New("unable to watch pod " + s.name)
			}
		case <-time.After(30 * time.Second):
			return "", errors.New("no state change after 30 seconds for pod " + s.name)
		}
	}

}

func (s *PodScraper) chooseContainer(p *v1.Pod) string {
	if p != nil {
		if len(p.Spec.Containers) == 1 {
			// Let Kubernetes auto-detect
			return ""
		}
		for _, c := range p.Spec.Containers {
			if _, ok := commonUserContainerNames[c.Name]; ok {
				return c.Name
			}
		}
	}
	return ""
}
