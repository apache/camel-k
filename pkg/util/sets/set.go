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

package sets

// Set is a very basic implementation for a Set data structure.
type Set struct {
	keys map[string]bool
}

// NewSet creates an empty Set.
func NewSet() *Set {
	return &Set{
		keys: make(map[string]bool),
	}
}

// Union creates a Set from the union of two.
func Union(s1, s2 *Set) *Set {
	items1 := s1.List()
	items2 := s2.List()

	unionSet := NewSet()
	unionSet.Add(items1...)
	unionSet.Add(items2...)

	return unionSet
}

// List returns the list of items of the Set.
func (s *Set) List() []string {
	keys := make([]string, len(s.keys))

	i := 0
	for k := range s.keys {
		keys[i] = k
		i++
	}

	return keys
}

// Add items to the Set.
func (s *Set) Add(items ...string) {
	for _, i := range items {
		s.keys[i] = true
	}
}

// Merge with items from another set.
func (s *Set) Merge(another *Set) {
	s.Add(another.List()...)
}

// Each traverses the items in the Set, calling the provided function for each
// Set member. Traversal will continue until all items in the Set have been
// visited, or if the closure returns false.
func (s *Set) Each(f func(item string) bool) {
	for item := range s.keys {
		if !f(item) {
			break
		}
	}
}

// Has returns true if it contains the item.
func (s *Set) Has(item string) bool {
	return s.keys[item]
}

// IsEmpty returns true if the set has no items.
func (s *Set) IsEmpty() bool {
	return s.keys == nil || len(s.keys) == 0
}

// Size returns the number if items in the Set.
func (s *Set) Size() int {
	return len(s.keys)
}
