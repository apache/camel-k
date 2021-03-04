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

package event

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
)

type sinkLessBroadcaster struct {
	broadcaster record.EventBroadcaster
}

func (s sinkLessBroadcaster) StartEventWatcher(eventHandler func(*corev1.Event)) watch.Interface {
	return s.broadcaster.StartEventWatcher(eventHandler)
}

func (s sinkLessBroadcaster) StartRecordingToSink(sink record.EventSink) watch.Interface {
	return watch.NewEmptyWatch()
}

func (s sinkLessBroadcaster) StartLogging(logf func(format string, args ...interface{})) watch.Interface {
	return s.broadcaster.StartLogging(logf)
}

func (s sinkLessBroadcaster) StartStructuredLogging(verbosity klog.Level) watch.Interface {
	return s.broadcaster.StartStructuredLogging(verbosity)
}

func (s sinkLessBroadcaster) NewRecorder(scheme *runtime.Scheme, source corev1.EventSource) record.EventRecorder {
	return s.broadcaster.NewRecorder(scheme, source)
}

func (s sinkLessBroadcaster) Shutdown() {
	s.broadcaster.Shutdown()
}

var _ record.EventBroadcaster = &sinkLessBroadcaster{}

func NewSinkLessBroadcaster(broadcaster record.EventBroadcaster) record.EventBroadcaster {
	return &sinkLessBroadcaster{
		broadcaster: broadcaster,
	}
}
