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

package knative

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelUri(t *testing.T) {
	assert.Equal(t, "pippo", ExtractChannelName("knative:channel/pippo"))
	assert.Equal(t, "pippo1", ExtractChannelName("knative://channel/pippo1"))
	assert.Equal(t, "pippo-2", ExtractChannelName("knative://channel/pippo-2?pluto=12"))
	assert.Equal(t, "pip-p-o", ExtractChannelName("knative:/channel/pip-p-o?pluto=12"))
	assert.Equal(t, "pip.po", ExtractChannelName("knative:channel/pip.po?pluto=12"))
	assert.Equal(t, "pip.po-1", ExtractChannelName("knative:channel/pip.po-1/hello"))
	assert.Empty(t, ExtractChannelName("http://wikipedia.org"))
	assert.Empty(t, ExtractChannelName("a:knative:channel/chan"))
	assert.Empty(t, ExtractChannelName("knative:channel/pippa$"))
}
