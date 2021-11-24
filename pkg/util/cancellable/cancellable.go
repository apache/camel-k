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

package cancellable

import "context"

// A Context with cancellation trait.
type Context interface {
	context.Context

	Cancel()
}

// NewContext returns an empty cancelable Context.
func NewContext() Context {
	return NewContextWithParent(context.TODO())
}

// NewContextWithParent returns an empty cancelable Context with a parent.
func NewContextWithParent(parent context.Context) Context {
	c, cc := context.WithCancel(parent)

	return &cancellableContext{
		Context: c,
		cancel:  cc,
	}
}

type cancellableContext struct {
	context.Context
	cancel func()
}

func (c *cancellableContext) Cancel() {
	c.cancel()
}
