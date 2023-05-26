/* A re-implementation of the amazing imapgrap in plain Golang.
Copyright (C) 2022  Torsten Long

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

func TestRangeCannonicalizationSuccess(t *testing.T) {
	testRanges := []struct {
		orgRange   rangeT
		canonRange rangeT
	}{
		{
			orgRange:   rangeT{start: 4, end: 5},
			canonRange: rangeT{start: 4, end: 5},
		},
		{
			orgRange:   rangeT{start: -8, end: -5},
			canonRange: rangeT{start: 3, end: 6},
		},
		{
			orgRange:   rangeT{start: -2, end: 0},
			canonRange: rangeT{start: 9, end: 11},
		},
	}

	orgRanges := []rangeT{}
	canonRanges := []rangeT{}
	for _, r := range testRanges {
		orgRanges = append(orgRanges, r.orgRange)
		canonRanges = append(canonRanges, r.canonRange)
	}

	actualCanonRanges, err := canonicalizeRanges(orgRanges, 1, 11)

	assert.NoError(t, err)
	assert.Equal(t, canonRanges, actualCanonRanges)
}

func TestRangeCannonicalizationFailure(t *testing.T) {
	testRanges := []rangeT{
		// Start larger than end.
		{start: 8, end: 4},
		// End larger than maximum possible.
		{start: 1, end: 100},
		// Start smaller than minimum possible.
		{start: 0, end: 10},
	}

	for _, r := range testRanges {
		_, err := canonicalizeRanges([]rangeT{r}, 1, 11)
		assert.Error(t, err)
	}
}

func TestRangeAccumulation(t *testing.T) {
	ranges := []rangeT{
		{start: 0, end: 10},
		{start: 20, end: 31},
		{start: 42, end: 53},
	}

	sum := accumulateRanges(ranges)

	assert.Equal(t, 32, sum)
}
