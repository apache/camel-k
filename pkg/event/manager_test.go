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
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// fakeRecorder implements events.EventRecorder and captures calls.
type fakeRecorder struct {
	called    bool
	obj       runtime.Object
	eventtype string
	reason    string
	action    string
	message   string
}

func (f *fakeRecorder) Eventf(obj runtime.Object, old runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
	f.called = true
	f.obj = obj
	f.eventtype = eventtype
	f.reason = reason
	f.action = action
	f.message = fmt.Sprintf(note, args...)
}

// Unused but required to satisfy interface
func (f *fakeRecorder) Event(obj runtime.Object, old runtime.Object, eventtype, reason, action, note string) {
}

func TestNotifyError(t *testing.T) {
	err := fmt.Errorf("boom")

	oldObj := &runtime.Unknown{}
	newObj := &runtime.Unknown{}

	tests := []struct {
		name         string
		old          runtime.Object
		new          runtime.Object
		expectCalled bool
		expectedObj  runtime.Object
	}{
		{
			name:         "uses newResource when not nil",
			old:          oldObj,
			new:          newObj,
			expectCalled: true,
			expectedObj:  newObj,
		},
		{
			name:         "falls back to old when newResource is nil",
			old:          oldObj,
			new:          nil,
			expectCalled: true,
			expectedObj:  oldObj,
		},
		{
			name:         "does nothing when both are nil",
			old:          nil,
			new:          nil,
			expectCalled: false,
			expectedObj:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &fakeRecorder{}

			NotifyError(rec, tt.old, tt.new, "my-name", "MyKind", err)

			if tt.expectCalled != rec.called {
				t.Fatalf("expected called=%v, got %v", tt.expectCalled, rec.called)
			}

			if !tt.expectCalled {
				return
			}

			if rec.obj != tt.expectedObj {
				t.Errorf("expected object %v, got %v", tt.expectedObj, rec.obj)
			}

			if rec.eventtype != corev1.EventTypeWarning {
				t.Errorf("expected event type %s, got %s", corev1.EventTypeWarning, rec.eventtype)
			}

			if rec.reason != "MyKindError" {
				t.Errorf("unexpected reason: %s", rec.reason)
			}

			if rec.action != "MyKindReconciliation" {
				t.Errorf("unexpected action: %s", rec.action)
			}

			expectedMsg := "Cannot reconcile MyKind my-name: boom"
			if rec.message != expectedMsg {
				t.Errorf("expected message %q, got %q", expectedMsg, rec.message)
			}
		})
	}
}
