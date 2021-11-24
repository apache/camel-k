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

// Package sync provides useful tools to get notified when a file system resource changes
package sync

import (
	"context"
	"time"

	"github.com/apache/camel-k/pkg/util/log"
	"github.com/radovskyb/watcher"
)

// File returns a channel that signals each time the content of the file changes.
func File(ctx context.Context, path string) (<-chan bool, error) {
	w := watcher.New()
	if err := w.Add(path); err != nil {
		return nil, err
	}
	w.FilterOps(watcher.Write)

	out := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.Event:
				out <- true
			}
		}
	}()

	go func() {
		if err := w.Start(200 * time.Millisecond); err != nil {
			log.Error(err, "Error while starting watcher")
			close(out)
		}
	}()

	return out, nil
}
