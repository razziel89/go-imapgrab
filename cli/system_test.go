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
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func catchStdoutStderr(t *testing.T) func() (string, string) {
	t.Helper()

	// Automatically restore the old streams.
	orgStdout := os.Stdout
	orgStderr := os.Stderr
	t.Cleanup(func() {
		os.Stdout = orgStdout
		os.Stderr = orgStderr
		// Getting the current output for the log module is not possible. Thus, at the end of the
		// test, we redirect logging to stderr, which it was originally.
		log.SetOutput(orgStderr)
	})

	// Create a temporary file that will contain the new stdout and redirect.
	fakeStdout := filepath.Join(t.TempDir(), "stdout")
	stdout, err := os.Create(fakeStdout) //nolint:gosec
	require.NoError(t, err)
	os.Stdout = stdout

	// Create a temporary file that will contain the new stderr and redirect.
	fakeStderr := filepath.Join(t.TempDir(), "stderr")
	stderr, err := os.Create(fakeStderr) //nolint:gosec
	require.NoError(t, err)
	os.Stderr = stderr
	log.SetOutput(stderr)

	t.Cleanup(func() {
		err := stdout.Close()
		require.NoError(t, err)
		err = stderr.Close()
		require.NoError(t, err)
	})

	// Create a function that, when called, reads the current values written to stdout and stderr
	// and returns them. As we catch stdout and stderr, which are supposed to be human-readable, a
	// string is the suitable return type instead of []byte.
	readStdouterr := func() (string, string) {
		stdout, err := os.ReadFile(fakeStdout) //nolint:gosec
		require.NoError(t, err)
		stderr, err := os.ReadFile(fakeStderr) //nolint:gosec
		require.NoError(t, err)
		return string(stdout), string(stderr)
	}

	return readStdouterr
}

func waitUntilConnected(t *testing.T, addr string) bool {
	t.Helper()
	// Give the server time to come up. Sadly, there is no way to actually detect whether the server
	// is up other than connecting to it. Thus, we simply try to connect every now and again and
	// sleep for some time in between.
	connected := false
	for try := 0; !connected && try < 100; try++ {
		time.Sleep(time.Millisecond)
		client, err := client.Dial(addr)
		if err == nil {
			connected = true
			err := client.Logout()
			require.NoError(t, err)
		}
	}
	return connected
}

// Set up a fake, in-memory mail server that has exactly one mailbox "INBOX" for a user with user
// name "username" and password "password". That one mailbox contains exactly one email.
func setUpFakeServerAndCommand(
	t *testing.T, args []string,
) (func() error, func() (string, string)) {
	t.Helper()
	server := server.New(memory.New())
	// Allow unauthenticated connections for testing.
	server.AllowInsecureAuth = true
	// Listen on a high local port and only on locahost. This is a test server, which means we
	// should not listen on all interfaces.
	server.Addr = "127.0.0.1:30218"

	// Have server listen in separate goroutine to be able to handle requests asyncronously. The
	// channel is used to ensure the server stops listening before the main goroutine finishes
	// execution.
	syncChan := make(chan bool)
	var serverErr error
	go func() {
		serverErr = server.ListenAndServe()
		syncChan <- true
	}()
	if !waitUntilConnected(t, server.Addr) {
		t.Fatal("cannot connect to the fake server in time")
	}

	var rootConf rootConfigT
	var downloadConf downloadConfigT

	var cmd *cobra.Command
	switch args[0] {
	case "root":
		cmd = getRootCmd()
	case "list":
		// Always disable the keyring by making this a test run.
		cmd = getListCmd(&rootConf, nil, false, &corer{})
	case "download":
		// Always disable the keyring by making this a test run.
		cmd = getDownloadCmd(&rootConf, &downloadConf, nil, false, &corer{}, lock)
		initDownloadFlags(cmd, &downloadConf)
	}
	// All commands use the root flags.
	initRootFlags(cmd, &rootConf)

	// Make sure the arguments used for the test run are known to the command.
	err := cmd.ParseFlags(args)
	require.NoError(t, err)

	stdouterrGetter := catchStdoutStderr(t)

	cleanup := func() {
		err := server.Close()
		require.NoError(t, err)
		<-syncChan
		if serverErr != nil {
			require.ErrorContains(t, serverErr, "use of closed network connection")
		}
	}
	t.Cleanup(cleanup)

	return cmd.Execute, stdouterrGetter
}

func TestSystemListSuccess(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	args := []string{"list", "--server=127.0.0.1", "--port=30218", "--user=username", "-v"}

	execute, stdouterr := setUpFakeServerAndCommand(t, args)

	err := execute()
	assert.NoError(t, err)
	stdout, stderr := stdouterr()
	assert.Equal(t, "INBOX\n", stdout)
	assert.Contains(t, stderr, "INFO retrieving folders")
	assert.Contains(t, stderr, "INFO retrieved 1 folders")
}

func TestSystemListAuthError(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	args := []string{"list", "--server=127.0.0.1", "--port=30218", "--user=something-else", "-v"}

	execute, stdouterr := setUpFakeServerAndCommand(t, args)

	err := execute()
	assert.ErrorContains(t, err, "Bad username or password")
	stdout, stderr := stdouterr()
	assert.Equal(t, "\n", stdout)
	assert.Contains(t, stderr, "ERROR cannot log in")
}

func TestSystemDownloadSuccess(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")
	maildir := t.TempDir()

	args := []string{
		"download", "--server=127.0.0.1", "--port=30218", "--user=username", "-v", "--folder=_ALL_",
		"--path", maildir,
	}

	execute, stdouterr := setUpFakeServerAndCommand(t, args)

	err := execute()
	assert.NoError(t, err)
	stdout, stderr := stdouterr()
	assert.Equal(t, "", stdout)
	assert.Equal(t, "", stderr)

	// Ensure that the maildir looks as expected.
	// TODO
}
