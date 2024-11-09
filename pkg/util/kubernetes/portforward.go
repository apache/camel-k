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

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func PortForward(ctx context.Context, c client.Client, ns, labelSelector string, localPort, remotePort uint, stdOut, stdErr io.Writer) error {
	log.InitForCmd()
	var forwardPod *corev1.Pod
	forwardCtx, forwardCtxCancel := context.WithCancel(ctx)
	defer forwardCtxCancel()

	setupPortForward := func(pod *corev1.Pod) error {
		if forwardPod == nil && podReady(pod) {
			forwardPod = pod
			log.Debugf("Setting up Port Forward for pod with name: %q\n", forwardPod.Name)
			if _, err := portFowardPod(forwardCtx, c.GetConfig(), ns, forwardPod.Name, localPort, remotePort, stdOut, stdErr); err != nil {
				return err
			}
		}
		return nil
	}

	log.Debugf("First attempt to bootstrap Port Forward with LabelSelector: %v\n", labelSelector)
	list, err := bootstrapPortForward(ctx, c, ns, labelSelector, setupPortForward)
	if err != nil {
		return err
	}

	log.Debugf("Instantiating pod event watcher with LabelSelector: %v and ResourceVersion: %v in namespace: %v\n", labelSelector, list.ResourceVersion, ns)
	watcher, err := c.CoreV1().Pods(ns).Watch(ctx, metav1.ListOptions{
		LabelSelector:   labelSelector,
		ResourceVersion: list.ResourceVersion,
	})
	if err != nil {
		return err
	}

	events := watcher.ResultChan()

	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-events:
			if !ok {
				return nil
			}

			switch e.Type {
			case watch.Added:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					return fmt.Errorf("type assertion failed: %v", e.Object)
				}
				log.Debugf("Handling watch.Added event for pod with name: %v\n", pod.Name)
				if err := setupPortForward(pod); err != nil {
					return err
				}
			case watch.Modified:
				pod, ok := e.Object.(*corev1.Pod)
				if !ok {
					return fmt.Errorf("type assertion failed: %v", e.Object)
				}
				log.Debugf("Handling watch.Modified event for pod with name: %v\n", pod.Name)
				if err := setupPortForward(pod); err != nil {
					return err
				}
			case watch.Deleted:
				log.Debugf("Handling watch.Deleted event\n")
				if forwardPod != nil && e.Object != nil {
					deletedPod, ok := e.Object.(*corev1.Pod)
					if !ok {
						return fmt.Errorf("type assertion failed: %v", e.Object)
					}
					log.Debugf("Handling watch.Deleted event for pod with name: %v while Port Forward was active for pod with name: %v\n", deletedPod.Name, forwardPod.Name)
					if deletedPod.Name == forwardPod.Name {
						forwardPod = nil

						log.Debugf("Handling watch.Deleted event, since the pod with Port Forward enabled has been deleted we try to bootstrap Port Forward with LabelSelector: %v\n", labelSelector)
						_, err := bootstrapPortForward(ctx, c, ns, labelSelector, setupPortForward)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
}

func bootstrapPortForward(ctx context.Context, c client.Client, ns string, labelSelector string, setupPortForward func(pod *corev1.Pod) error) (*corev1.PodList, error) {
	list, err := c.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	if len(list.Items) > 0 {
		log.Debugf("Bootstrapping Port Forward for pod with name: %v\n", list.Items[0].Name)
		if err := setupPortForward(&list.Items[0]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func portFowardPod(ctx context.Context, config *restclient.Config, ns, pod string, localPort, remotePort uint, stdOut, stdErr io.Writer) (string, error) {
	c, err := corev1client.NewForConfig(config)
	if err != nil {
		return "", err
	}

	url := c.RESTClient().Post().
		Resource("pods").
		Namespace(ns).
		Name(pod).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, stdOut, stdErr)
	if err != nil {
		return "", err
	}

	go func() {
		// Start the port forwarder
		err = forwarder.ForwardPorts()
		if err != nil {
			log.Errorf(err, "error while forwarding ports")
		}
	}()

	go func() {
		// Stop the port forwarder when the context ends
		<-ctx.Done()
		close(stopChan)
	}()

	select {
	case <-readyChan:
		ports, err := forwarder.GetPorts()
		if err != nil {
			return "", err
		}
		if len(ports) != 1 {
			return "", errors.New("wrong ports opened")
		}
		return fmt.Sprintf("localhost:%d", ports[0].Local), nil
	case <-ctx.Done():
		return "", errors.New("context closed")
	}
}

func podReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
