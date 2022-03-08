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

import "fmt"

// Type rangeT describes a range of integer values. Usually, the range includes start but excludes
// end.
type rangeT struct {
	start int
	end   int
}

func canonicalizeRange(r rangeT, start, end int) (rangeT, error) {
	// Convert negative indices to count backwards from end.
	if r.start < 0 {
		r.start = end + r.start
		// Handle special case in which the range -n,0 has been given, with n being a positive
		// integer. In this case, the end has to be interpreted as the last message. All other cases
		// require no special handling.
		if r.end == 0 {
			r.end = end
		}
	}
	if r.end < 0 {
		r.end = end + r.end
	}
	// Make sure the range's end is larger than its start.
	if !(r.end > r.start) {
		return r, fmt.Errorf("range end must be larger than range start")
	}
	// Make sure the range's values do not exceed the available range.
	if r.start < start {
		return r, fmt.Errorf("range start cannot be smaller than %d", start)
	}
	if r.end > end {
		return r, fmt.Errorf("range end cannot be larger than %d", end)
	}
	return r, nil
}

func accumulateRanges(ranges []rangeT) int {
	total := 0

	for _, r := range ranges {
		total += r.end - r.start
	}

	return total
}
