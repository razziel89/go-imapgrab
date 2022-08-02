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
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TODO: implement mock for coreOps

func doTestOfDownloadOrList(
	t *testing.T,
	getCmdFn func(
		rootConf *rootConfigT, keyring keyringOps, prodRun bool, ops coreOps,
	) *cobra.Command,
) {
	t.Setenv("IGRAB_PASSWORD", "some password")

	mk := &mockKeyring{}

	rootConf := rootConfigT{}
	cmd := getCmdFn(&rootConf, mk, false)

	err := cmd.Execute()
	assert.Error(t, err)
}

func doTestOfDownloadOrListNoKeyringProdRun(
	t *testing.T,
	getCmdFn func(rootConf *rootConfigT, keyring keyringOps, prodRun bool) *cobra.Command,
) {
	if orgVal, found := os.LookupEnv("IGRAB_PASSWORD"); found {
		defer func() {
			err := os.Setenv("IGRAB_PASSWORD", orgVal)
			assert.NoError(t, err)
		}()
	}
	err := os.Unsetenv("IGRAB_PASSWORD")
	assert.NoError(t, err)

	mk := &mockKeyring{}

	rootConf := rootConfigT{}
	cmd := getCmdFn(&rootConf, mk, true)

	// The keyring is disabled via user flags, which are evaluated after the command has been
	// constructed.
	orgNoKeyring := noKeyring
	noKeyring = true
	t.Cleanup(func() { noKeyring = orgNoKeyring })

	err = cmd.Execute()
	assert.Error(t, err)
}

func TestDownloadCommand(t *testing.T) {
	doTestOfDownloadOrList(t, getDownloadCmd)
}

func TestDownloadCommandNoKeyringProdRun(t *testing.T) {
	doTestOfDownloadOrListNoKeyringProdRun(t, getDownloadCmd)
}
