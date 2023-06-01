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
	"testing"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Set up a fake, in-memory mail server that has exactly one mailbox "INBOX" for a user with user
// name "username" and password "password". That one mailbox contains exactly one email.
func setUpFakeServerAndCommand(t *testing.T, args []string) (func() error, func()) {
	addr := "127.0.0.1:30218"

	backend := memory.New()
	srv := server.New(backend)

	// Allow unauthenticated connections for testing.
	srv.AllowInsecureAuth = true
	// Listen on a high local port and only on locahost. This is a test server, which means we
	// should not listen on all interfaces.
	srv.Addr = addr

	// Have server listen in separate goroutine to be able to handle requests asyncronously.
	go func() {
		err := srv.ListenAndServe()
		require.NoError(t, err)
	}()

	var rootConf rootConfigT
	var downloadConf downloadConfigT

	var cmd *cobra.Command
	switch args[0] {
	case "root":
		cmd = getRootCmd()
		initRootFlags(cmd, &rootConf)
	case "list":
		// Always disable the keyring by making this a test run.
		cmd = getListCmd(&rootConf, nil, false, &corer{})
		initRootFlags(cmd, &rootConf)
	case "download":
		// Always disable the keyring by making this a test run.
		cmd = getDownloadCmd(&rootConf, &downloadConf, nil, false, &corer{}, lock)
		initRootFlags(cmd, &rootConf)
		initDownloadFlags(cmd, &downloadConf)
	}

	// Make sure the arguments used for the test run are known to the command.
	err := cmd.ParseFlags(args)
	require.NoError(t, err)

	execute := func() error {
		return cmd.Execute()
	}

	cleanup := func() {
		err := srv.Close()
		require.NoError(t, err)
	}

	return execute, cleanup
}

func TestSystemSuccess(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	args := []string{
		"list", "--server=127.0.0.1", "--port=30218", "--user=username", "-v",
	}

	execute, cleanup := setUpFakeServerAndCommand(t, args)
	defer cleanup()

	err := execute()
	assert.NoError(t, err)
}
