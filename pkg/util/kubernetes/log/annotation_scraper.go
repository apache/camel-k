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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/multierr"

	klog "github.com/apache/camel-k/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

// SelectorScraper scrapes all pods with a given selector.
type SelectorScraper struct {
	client               kubernetes.Interface
	namespace            string
	defaultContainerName string
	labelSelector        string
	podScrapers          sync.Map
	counter              uint64
	L                    klog.Logger
}

// NewSelectorScraper creates a new SelectorScraper.
func NewSelectorScraper(client kubernetes.Interface, namespace string, defaultContainerName string,
	labelSelector string) *SelectorScraper {
	return &SelectorScraper{
		client:               client,
		namespace:            namespace,
		defaultContainerName: defaultContainerName,
		labelSelector:        labelSelector,
		L:                    klog.WithName("scraper").WithName("label").WithValues("selector", labelSelector),
	}
}

// Start returns a reader that streams the log of all selected pods.
func (s *SelectorScraper) Start(ctx context.Context) *bufio.Reader {
	pipeIn, pipeOut := io.Pipe()
	bufPipeIn := bufio.NewReader(pipeIn)
	bufPipeOut := bufio.NewWriter(pipeOut)
	closeFun := func() error {
		return multierr.Append(
			bufPipeOut.Flush(),
			pipeOut.Close())
	}
	go s.periodicSynchronize(ctx, bufPipeOut, closeFun)
	return bufPipeIn
}

func (s *SelectorScraper) periodicSynchronize(ctx context.Context, out *bufio.Writer, clientCloser func() error) {
	if err := s.synchronize(ctx, out); err != nil {
		s.L.Info("Could not synchronize log")
	}
	select {
	case <-ctx.Done():
		// cleanup
		s.podScrapers.Range(func(_, v interface{}) bool {
			if canc, isCanc := v.(context.CancelFunc); isCanc {
				canc()
			}

			return true
		})
		if err := clientCloser(); err != nil {
			s.L.Error(err, "Unable to close the client")
		}
	case <-time.After(2 * time.Second):
		go s.periodicSynchronize(ctx, out, clientCloser)
	}
}

func (s *SelectorScraper) synchronize(ctx context.Context, out *bufio.Writer) error {
	list, err := s.listPods(ctx)
	if err != nil {
		return err
	}

	present := make(map[string]bool)
	for _, pod := range list.Items {
		present[pod.Name] = true
		if _, ok := s.podScrapers.Load(pod.Name); !ok {
			s.addPodScraper(ctx, pod.Name, out)
		}
	}

	toBeRemoved := make(map[string]bool)
	s.podScrapers.Range(func(k, _ interface{}) bool {
		if str, isStr := k.(string); isStr {
			if _, ok := present[str]; !ok {
				toBeRemoved[str] = true
			}
		}

		return true
	})

	for podName := range toBeRemoved {
		if scr, ok := s.podScrapers.Load(podName); ok {
			if canc, ok2 := scr.(context.CancelFunc); ok2 {
				canc()
				s.podScrapers.Delete(podName)
			}
		}
	}
	return nil
}

func (s *SelectorScraper) addPodScraper(ctx context.Context, podName string, out *bufio.Writer) {
	podScraper := NewPodScraper(s.client, s.namespace, podName, s.defaultContainerName)
	podCtx, podCancel := context.WithCancel(ctx)
	id := atomic.AddUint64(&s.counter, 1)
	prefix := "[" + strconv.FormatUint(id, 10) + "] "
	podReader := podScraper.Start(podCtx)
	s.podScrapers.Store(podName, podCancel)
	go func() {
		defer podCancel()

		if _, err := out.WriteString(prefix + "Monitoring pod " + podName + "\n"); err != nil {
			s.L.Error(err, "Cannot write to output")
			return
		}
		for {
			str, err := podReader.ReadString('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				s.L.Error(err, "Cannot read from pod stream")
				return
			}
			if _, err := out.WriteString(prefix + str); err != nil {
				s.L.Error(err, "Cannot write to output")
				return
			}
			if err := out.Flush(); err != nil {
				s.L.Error(err, "Cannot flush output")
				return
			}
			if podCtx.Err() != nil {
				return
			}
		}
	}()
}

func (s *SelectorScraper) listPods(ctx context.Context) (*corev1.PodList, error) {
	list, err := s.client.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: s.labelSelector,
	})
	if err != nil {
		return nil, err
	}

	return list, nil
}
