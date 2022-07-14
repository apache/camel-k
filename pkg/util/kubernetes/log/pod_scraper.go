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

	"go.uber.org/multierr"

	klog "github.com/apache/camel-k/pkg/util/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

// PodScraper scrapes logs of a specific pod.
type PodScraper struct {
	namespace            string
	podName              string
	defaultContainerName string
	client               kubernetes.Interface
	L                    klog.Logger
}

// NewPodScraper creates a new pod scraper.
func NewPodScraper(c kubernetes.Interface, namespace string, podName string, defaultContainerName string) *PodScraper {
	return &PodScraper{
		namespace:            namespace,
		podName:              podName,
		defaultContainerName: defaultContainerName,
		client:               c,
		L:                    klog.WithName("scraper").WithName("pod").WithValues("name", podName),
	}
}

// Start returns a reader that streams the pod logs.
func (s *PodScraper) Start(ctx context.Context) *bufio.Reader {
	pipeIn, pipeOut := io.Pipe()
	bufPipeIn := bufio.NewReader(pipeIn)
	bufPipeOut := bufio.NewWriter(pipeOut)
	closeFun := func() error {
		return multierr.Append(
			bufPipeOut.Flush(),
			pipeOut.Close())
	}
	go s.doScrape(ctx, bufPipeOut, closeFun)
	return bufPipeIn
}

func (s *PodScraper) doScrape(ctx context.Context, out *bufio.Writer, clientCloser func() error) {
	containerName, err := s.waitForPodRunning(ctx, s.namespace, s.podName, s.defaultContainerName)
	if err != nil {
		s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
		return
	}
	logOptions := corev1.PodLogOptions{
		Follow:    true,
		Container: containerName,
	}
	byteReader, err := s.client.CoreV1().Pods(s.namespace).GetLogs(s.podName, &logOptions).Stream(ctx)
	if err != nil {
		s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
		return
	}

	reader := bufio.NewReader(byteReader)
	for {
		data, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			break
		}
		if _, err = out.Write(data); err != nil {
			break
		}

		if err = out.Flush(); err != nil {
			break
		}
	}

	s.handleAndRestart(ctx, err, 5*time.Second, out, clientCloser)
}

func (s *PodScraper) handleAndRestart(ctx context.Context, err error, wait time.Duration, out *bufio.Writer, clientCloser func() error) {
	if err != nil {
		s.L.Error(err, "error caught during log scraping")
	}

	if ctx.Err() != nil {
		s.L.Debug("Pod will no longer be monitored")
		if err := clientCloser(); err != nil {
			s.L.Error(err, "Unable to close the client")
		}
		return
	}

	s.L.Debugf("Retrying to scrape pod logs in %f seconds...", wait.Seconds())
	select {
	case <-time.After(wait):
		break
	case <-ctx.Done():
		if err := clientCloser(); err != nil {
			s.L.Error(err, "Unable to close the client")
		}
		return
	}

	s.doScrape(ctx, out, clientCloser)
}

// waitForPodRunning waits for a given pod to reach the running state.
// It may return the internal container to watch if present.
func (s *PodScraper) waitForPodRunning(ctx context.Context, namespace string, podName string, defaultContainerName string) (string, error) {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
	}
	podClient := s.client.CoreV1().Pods(pod.Namespace)
	watcher, err := podClient.Watch(ctx, metav1.ListOptions{
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
				var recvPod *corev1.Pod
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
				} else if gotPod, ok := e.Object.(*corev1.Pod); ok {
					recvPod = gotPod
				}

				if recvPod != nil && recvPod.Status.Phase == corev1.PodRunning {
					return s.chooseContainer(recvPod, defaultContainerName), nil
				}
			} else if e.Type == watch.Deleted || e.Type == watch.Error {
				return "", errors.New("unable to watch pod " + s.podName)
			}
		case <-time.After(30 * time.Second):
			return "", errors.New("no state change after 30 seconds for pod " + s.podName)
		}
	}
}

func (s *PodScraper) chooseContainer(p *corev1.Pod, defaultContainerName string) string {
	if p != nil {
		if len(p.Spec.Containers) == 1 {
			// Let Kubernetes auto-detect
			return ""
		}
		// Fallback to first container name
		containerNameFound := p.Spec.Containers[0].Name
		for _, c := range p.Spec.Containers {
			if _, ok := commonUserContainerNames[c.Name]; ok {
				return c.Name
			} else if c.Name == defaultContainerName {
				containerNameFound = defaultContainerName
			}
		}
		return containerNameFound
	}
	return ""
}
