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

package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

type compositeKameletRepository struct {
	repositories []KameletRepository
}

var _ KameletRepository = &compositeKameletRepository{}

func newCompositeKameletRepository(repositories ...KameletRepository) KameletRepository {
	return &compositeKameletRepository{
		repositories: repositories,
	}
}

func (c compositeKameletRepository) List(ctx context.Context) ([]string, error) {
	kSet := make(map[string]bool)
	for _, repo := range c.repositories {
		lst, err := repo.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, kam := range lst {
			kSet[kam] = true
		}
	}
	res := make([]string, 0, len(kSet))
	for kam := range kSet {
		res = append(res, kam)
	}
	sort.Strings(res)
	return res, nil
}

func (c compositeKameletRepository) Get(ctx context.Context, name string) (*v1alpha1.Kamelet, error) {
	for _, repo := range c.repositories {
		kam, err := repo.Get(ctx, name)
		if kam != nil || err != nil {
			return kam, err
		}
	}
	return nil, nil
}

func (c *compositeKameletRepository) String() string {
	descs := make([]string, 0, len(c.repositories))
	for _, repo := range c.repositories {
		descs = append(descs, repo.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(descs, ", "))
}
