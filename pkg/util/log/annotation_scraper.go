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

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SelectorScraper scrapes all pods with a given selector
type SelectorScraper struct {
	namespace     string
	labelSelector string
	podScrapers   sync.Map
	counter       uint64
}

// NewSelectorScraper creates a new SelectorScraper
func NewSelectorScraper(namespace string, labelSelector string) *SelectorScraper {
	return &SelectorScraper{
		namespace:     namespace,
		labelSelector: labelSelector,
	}
}

// Start returns a reader that streams the log of all selected pods
func (s *SelectorScraper) Start(ctx context.Context) *bufio.Reader {
	pipeIn, pipeOut := io.Pipe()
	bufPipeIn := bufio.NewReader(pipeIn)
	bufPipeOut := bufio.NewWriter(pipeOut)
	closeFun := func() error {
		bufPipeOut.Flush()
		return pipeOut.Close()
	}
	go s.periodicSynchronize(ctx, bufPipeOut, closeFun)
	return bufPipeIn
}

func (s *SelectorScraper) periodicSynchronize(ctx context.Context, out *bufio.Writer, clientCloser func() error) {
	err := s.synchronize(ctx, out, clientCloser)
	if err != nil {
		logrus.Warn("Could not synchronize log by label " + s.labelSelector)
	}
	select {
	case <-ctx.Done():
		// cleanup
		s.podScrapers.Range(func(k, v interface{}) bool {
			if canc, isCanc := v.(context.CancelFunc); isCanc {
				canc()
			}

			return true
		})
		clientCloser()
	case <-time.After(2 * time.Second):
		go s.periodicSynchronize(ctx, out, clientCloser)
	}
}

func (s *SelectorScraper) synchronize(ctx context.Context, out *bufio.Writer, clientCloser func() error) error {
	list, err := s.listPods()
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
	s.podScrapers.Range(func(k, v interface{}) bool {
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

func (s *SelectorScraper) addPodScraper(ctx context.Context, name string, out *bufio.Writer) {
	podScraper := NewPodScraper(s.namespace, name)
	podCtx, podCancel := context.WithCancel(ctx)
	id := atomic.AddUint64(&s.counter, 1)
	prefix := "[" + strconv.FormatUint(id, 10) + "] "
	podReader := podScraper.Start(podCtx)
	s.podScrapers.Store(name, podCancel)
	go func() {
		defer podCancel()

		out.WriteString(prefix + "Monitoring pod " + name)
		for {
			str, err := podReader.ReadString('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				logrus.Error("Cannot read from pod stream: ", err)
				return
			}
			out.WriteString(prefix + str)
			out.Flush()
			if podCtx.Err() != nil {
				return
			}
		}
	}()

}

func (s *SelectorScraper) listPods() (*v1.PodList, error) {
	list := v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
	}

	err := sdk.List(s.namespace, &list, sdk.WithListOptions(&metav1.ListOptions{
		LabelSelector: s.labelSelector,
	}))

	if err != nil {
		return nil, err
	}

	return &list, nil
}
