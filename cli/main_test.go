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

package main

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	calledRootCmd := false
	orgRootCmd := rootCmd
	t.Cleanup(func() { rootCmd = orgRootCmd })
	rootCmd = &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			calledRootCmd = true
			return fmt.Errorf("some error")
		},
	}

	calledLogFatal := false
	orgLogFatal := logFatal
	t.Cleanup(func() { logFatal = orgLogFatal })
	logFatal = func(_ ...interface{}) {
		calledLogFatal = true
	}

	main()

	assert.True(t, calledRootCmd)
	assert.True(t, calledLogFatal)
}
