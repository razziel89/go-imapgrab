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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOrderedSet(t *testing.T) {
	set := newOrderedSet(10)
	// Set shall be empty.
	assert.Equal(t, 0, set.len())
}

func TestSetFromSlice(t *testing.T) {
	set := setFromSlice([]string{"this", "is", "a", "slice"})
	assert.Equal(t, 4, set.len())
}

func TestSetAddRemoveHas(t *testing.T) {
	set := newOrderedSet(2)
	assert.Equal(t, 0, set.len())

	set.add("a string")
	assert.Equal(t, 1, set.len())
	assert.True(t, set.has("a string"))

	set.add("a string")
	assert.Equal(t, 1, set.len())

	set.remove("another string")
	assert.Equal(t, 1, set.len())

	set.remove("a string")
	assert.Equal(t, 0, set.len())
	assert.False(t, set.has("a string"))
}

func TestSetOrder(t *testing.T) {
	sli := []string{"this", "this", "is", "a", "slice", "is"}
	set := setFromSlice(sli)
	orderedSli := []string{"this", "is", "a", "slice"}

	assert.Equal(t, orderedSli, set.orderedEntries())
}

func TestSetOrderWithDelete(t *testing.T) {
	sli := []string{"this", "this", "is", "a", "slice", "is"}
	set := setFromSlice(sli)

	set.remove("is")
	set.remove("a")
	set.add("is")
	set.add("ordered")

	orderedSli := []string{"this", "slice", "is", "ordered"}

	assert.Equal(t, orderedSli, set.orderedEntries())
}

func TestIterator(t *testing.T) {
	sli := []string{"this", "this", "is", "a", "slice", "is"}
	set := setFromSlice(sli)

	expectedEntries := []string{"this", "is", "a", "slice"}

	for entry := range set.iterator() {
		found := false
		for _, check := range expectedEntries {
			if entry == check {
				found = true
				break
			}
		}
		assert.True(t, found)
	}
}

func TestUnion(t *testing.T) {
	largeSet := setFromSlice([]string{"this", "is", "a", "slice"})
	excludeMe := setFromSlice([]string{"this", "slice", "has", "other", "entries"})

	expectedUnion := []string{"is", "a"}
	union := largeSet.keepUnion(excludeMe)

	assert.Equal(t, expectedUnion, union)
}