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

package internal

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FakeRecorder implements events.EventRecorder and captures calls.
type FakeRecorder struct {
	Called    bool
	Obj       runtime.Object
	Eventtype string
	Reason    string
	Action    string
	Message   string
	Kind      schema.ObjectKind
}

func (f *FakeRecorder) Eventf(obj runtime.Object, old runtime.Object, eventtype, reason, action, note string, args ...any) {
	f.Called = true
	f.Obj = obj
	f.Eventtype = eventtype
	f.Reason = reason
	f.Action = action
	f.Message = fmt.Sprintf(note, args...)
	f.Kind = obj.GetObjectKind()
}

func (f *FakeRecorder) Event(obj runtime.Object, old runtime.Object, eventtype, reason, action, note string) {
}
