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

	"github.com/apache/camel-k/v2/pkg/internal"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
			rec := &internal.FakeRecorder{}

			NotifyError(rec, tt.old, tt.new, "my-name", "MyKind", err)

			if tt.expectCalled != rec.Called {
				t.Fatalf("expected called=%v, got %v", tt.expectCalled, rec.Called)
			}

			if !tt.expectCalled {
				return
			}

			if rec.Obj != tt.expectedObj {
				t.Errorf("expected object %v, got %v", tt.expectedObj, rec.Obj)
			}

			if rec.Eventtype != corev1.EventTypeWarning {
				t.Errorf("expected event type %s, got %s", corev1.EventTypeWarning, rec.Eventtype)
			}

			if rec.Reason != "MyKindError" {
				t.Errorf("unexpected reason: %s", rec.Reason)
			}

			if rec.Action != "MyKindReconciliation" {
				t.Errorf("unexpected action: %s", rec.Action)
			}

			expectedMsg := "Cannot reconcile MyKind my-name: boom"
			if rec.Message != expectedMsg {
				t.Errorf("expected message %q, got %q", expectedMsg, rec.Message)
			}
		})
	}
}
