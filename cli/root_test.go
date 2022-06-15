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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	rootCmd := getRootCmd()
	initRootFlags(rootCmd)
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestLogDebug(t *testing.T) {
	orgVerbose := verbose
	verbose = true
	t.Cleanup(func() { verbose = orgVerbose })

	// We deliberately do not catch output here.
	logDebug("some test", 123)
}
