// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/util/log"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/stretchr/testify/assert"
)

func TestPodLogScrape(t *testing.T) {
	token := "Hello Camel K!"
	pod, err := createDummyPod("scraped", "/bin/sh", "-c", "for i in `seq 1 50`; do echo \""+token+"\" && sleep 2; done")
	defer sdk.Delete(pod)
	assert.Nil(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	defer cancel()
	scraper := log.NewPodScraper(pod.Namespace, pod.Name)
	in := scraper.Start(ctx)

	res := make(chan bool)
	go func() {
		for {
			if dl, _ := ctx.Deadline(); time.Now().After(dl) {
				return
			}

			str, _ := in.ReadString('\n')
			if strings.Contains(str, token) {
				res <- true
				return
			}
		}
	}()

	select {
	case <-res:
		break
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timeout while waiting from token")
	}
}

func TestSelectorLogScrape(t *testing.T) {
	token := "Hello Camel K!"
	replicas := int32(3)
	deployment, err := createDummyDeployment("scraped-deployment", &replicas, "scrape", "me", "/bin/sh", "-c", "for i in `seq 1 50`; do echo \""+token+"\" && sleep 2; done")
	defer sdk.Delete(deployment)
	assert.Nil(t, err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
	defer cancel()
	scraper := log.NewSelectorScraper(deployment.Namespace, "scrape=me")
	in := scraper.Start(ctx)

	res := make(chan string)
	go func() {
		for {
			if dl, _ := ctx.Deadline(); time.Now().After(dl) {
				return
			}

			str, _ := in.ReadString('\n')
			if strings.Contains(str, token) {
				res <- str[0:3]
			}
		}
	}()

	recv := make(map[string]bool)
loop:
	for {
		select {
		case r := <-res:
			recv[r] = true
			if len(recv) == 3 {
				break loop
			}
		case <-time.After(13 * time.Second):
			assert.Fail(t, "timeout while waiting from token")
			break loop
		}
	}
}
