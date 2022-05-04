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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUniqueNameSuccess(t *testing.T) {
	// Use a set to ensure each name is really unique.
	set := map[string]struct{}{}
	hostname, err := os.Hostname()
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		currentDeliveryCount := deliveryCount

		newName, err := newUniqueName("")

		assert.NoError(t, err)
		assert.Greater(t, deliveryCount, currentDeliveryCount)
		assert.NotContains(t, set, newName)
		assert.Contains(t, newName, hostname)

		set[newName] = struct{}{}
	}
}

func TestNewUniqueNameFailure(t *testing.T) {
	currentDeliveryCount := deliveryCount

	_, err := newUniqueName("hostname with space breaks function")

	assert.Error(t, err)
	assert.Greater(t, deliveryCount, currentDeliveryCount)
}

func TestNewUniqueNameBrokenNameFixes(t *testing.T) {
	currentDeliveryCount := deliveryCount

	newName, err := newUniqueName("BrokenHostname/withSlash")

	assert.NoError(t, err)
	assert.Greater(t, deliveryCount, currentDeliveryCount)
	assert.Contains(t, newName, "BrokenHostname\\057withSlash")
}

func TestNewUniqueNameStartAndEnd(t *testing.T) {
	newName, err := newUniqueName("SomeHost")

	assert.NoError(t, err)
	// The following regex means:
	// - start with at least one digit followed by a dot
	// - end with dot followed by hostname
	// - contain the middle string with some information, see newUniqueName for details what the
	//   individual bits mean
	assert.Regexp(t, "^[0-9]+\\.M[0-9]+P[0-9]+Q[0-9]+R[a-fA-F0-9]+\\.SomeHost$", newName)
}
