/* A re-implementation of the amazing imapgrap in plain Golang.
Copyright (C) 2022  Torsten Sachse

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package core

import "sort"

// A simple ordered set type to simplify handling of lists of unique entries.
type orderedSet struct {
	data       map[string]int
	orderCount int
}

func newOrderedSet(expectedLength int) orderedSet {
	data := make(map[string]int, expectedLength)
	return orderedSet{
		data:       data,
		orderCount: 0,
	}
}

func setFromSlice(sli []string) orderedSet {
	s := newOrderedSet(len(sli))
	for _, value := range sli {
		s.add(value)
	}
	return s
}

func (s *orderedSet) add(key string) {
	if !s.has(key) {
		s.data[key] = s.orderCount
		s.orderCount++
	}
}

func (s *orderedSet) remove(key string) {
	delete(s.data, key)
}

func (s *orderedSet) has(key string) bool {
	_, found := s.data[key]
	return found
}

func (s *orderedSet) len() int {
	return len(s.data)
}

func (s *orderedSet) iterator() map[string]int {
	return s.data
}

type orderHelper struct {
	value string
	order int
}

// Obtain all entries in the order in which they were specified. This can be expensive if there
// are many entries. If the order of the keys is not important, rather use `iterator()` to obtain
// the raw data and extract the keys from there.
func (s *orderedSet) orderedEntries() []string {
	// First, extract all data in the form of (value, order) tuples.
	order := make([]orderHelper, 0, s.len())
	for key, keyOrder := range s.iterator() {
		order = append(order, orderHelper{key, keyOrder})
	}

	// Then, sort by order.
	lessFn := func(i, j int) bool {
		return order[i].order < order[j].order
	}
	sort.Slice(order, lessFn)

	// Then, extract values and return them.
	orderedKeysSli := make([]string, 0, s.len())
	for _, helper := range order {
		orderedKeysSli = append(orderedKeysSli, helper.value)
	}
	return orderedKeysSli
}

// Keep only those entries in the receiver set that are also in the other set. Return the entries
// that have been removed.
func (s *orderedSet) keepUnion(otherSet orderedSet) []string {
	removed := []string{}
	for entry := range s.iterator() {
		if !otherSet.has(entry) {
			s.remove(entry)
			removed = append(removed, entry)
		}
	}
	return removed
}
