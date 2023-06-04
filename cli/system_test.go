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
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultExpectedTestLogs = []string{
	"INFO connected",
	"INFO connecting to server 127.0.0.1",
	"INFO logged in",
	"INFO logging in as username with provided password",
	"INFO logging out",
	"INFO retrieved 1 folders",
	"INFO retrieving folders",
	"password taken from env var IGRAB_PASSWORD",
	"WARNING using insecure connection to locahost",
}

var defaultExpectedDownloadTestLogs = []string{
	"INFO all sub-directories found",
	"INFO appending to oldmail file",
	"INFO available folders are",
	"INFO checking for and reading oldmail file of possible maildir",
	"INFO checking for sub-directories of possible maildir",
	"INFO creating path to maildir",
	"INFO downloaded email",
	"INFO expanded to folders",
	"INFO expanding folder spec",
	"INFO found and read oldmail file",
	"INFO initializing maildir",
	"INFO moving email to permanent storage location",
	"INFO reading oldmail file",
	"INFO received information for 1 emails",
	"INFO retrieving information about emails stored on disk",
	"INFO retrieving information about emails stored on server",
	"INFO selected folder contains 1 emails",
	"INFO selecting folder:INBOX",
	"INFO there were 0/0/0 errors while: retrieving/delivering/remembering mail",
	"INFO will download 1 new emails",
	"INFO writing new email to file",
}

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

	tmpdir := t.TempDir()
	// Create a temporary file that will contain the new stdout and redirect.
	fakeStdout := filepath.Join(tmpdir, "stdout")
	stdout, err := os.Create(fakeStdout) //nolint:gosec
	require.NoError(t, err)
	os.Stdout = stdout

	// Create a temporary file that will contain the new stderr and redirect.
	fakeStderr := filepath.Join(tmpdir, "stderr")
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
func setUpFakeServerAndCommand(t *testing.T, args []string) func() error {
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
	require.True(t, waitUntilConnected(t, server.Addr), "cannot connect to test server in time")

	var rootConf rootConfigT
	var downloadConf downloadConfigT

	var cmd *cobra.Command
	switch args[0] {
	case "list":
		// Always disable the keyring by making this a test run.
		cmd = getListCmd(&rootConf, nil, false, &corer{})
	case "download":
		// Always disable the keyring by making this a test run.
		cmd = getDownloadCmd(&rootConf, &downloadConf, nil, false, &corer{}, lock)
	case "login":
		user, err := user.Current()
		require.NoError(t, err)
		mk := &mockKeyring{}
		mk.On("Set", "go-imapgrab/username@127.0.0.1:30218", user.Name, "password").Return(nil)
		t.Cleanup(func() { mk.AssertExpectations(t) })
		cmd = getLoginCmd(
			&rootConf, mk, func(int) ([]byte, error) { return []byte("password"), nil },
		)
	default:
		t.Log("unknown command")
		t.FailNow()
	}

	cmd.SetArgs(args)

	t.Cleanup(func() {
		err := server.Close()
		require.NoError(t, err)
		<-syncChan
		if serverErr != nil {
			require.ErrorContains(t, serverErr, "use of closed network connection")
		}
	})

	return cmd.Execute
}

func TestSystemListSuccess(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	args := []string{"list", "--server=127.0.0.1", "--port=30218", "--user=username", "-v"}
	stdouterr := catchStdoutStderr(t)
	execute := setUpFakeServerAndCommand(t, args)

	err := execute()

	assert.NoError(t, err)
	stdout, stderr := stdouterr()
	assert.Equal(t, "INBOX\n", stdout)
	for _, msg := range defaultExpectedTestLogs {
		assert.Contains(t, stderr, msg)
	}
}

func TestSystemListAuthError(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	args := []string{"list", "--server=127.0.0.1", "--port=30218", "--user=something-else", "-v"}
	stdouterr := catchStdoutStderr(t)
	execute := setUpFakeServerAndCommand(t, args)

	err := execute()

	assert.ErrorContains(t, err, "Bad username or password")
	stdout, stderr := stdouterr()
	assert.Equal(t, "\n", stdout)
	assert.Contains(t, stderr, "ERROR cannot log in")
}

func scanDirReplacingEmails(t *testing.T, dir string) (files []string, dirs []string) {
	host, err := os.Hostname()
	require.NoError(t, err)

	emailCount := 0

	getFilesAndDirs := func(path string, d fs.DirEntry, err error) error {
		// Return early on read errors.
		if err != nil {
			return err
		}
		// We treat all paths relative to the maildir.
		path = strings.TrimPrefix(strings.TrimPrefix(path, dir), string(os.PathSeparator))
		fmt.Println(path, d.Name())
		if d.IsDir() {
			dirs = append(dirs, path)
		} else {
			if strings.HasSuffix(path, fmt.Sprintf(".%s", host)) {
				// As the name of the actual email file will be pretty random, we replace the name
				// of the file by this placeholder, which is easier to check. This is a bit hacky
				// but means we do not have to mock the generation of the name. The file is
				// identified by ending on the hostname preceded by a dot.
				path = filepath.Join(filepath.Dir(path), fmt.Sprintf("email.%d", emailCount))
				emailCount++
			}
			files = append(files, path)
		}
		return nil
	}

	err = filepath.WalkDir(dir, getFilesAndDirs)
	require.NoError(t, err)

	return
}

func TestSystemDownloadSuccess(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "password")

	maildir := t.TempDir()
	args := []string{
		"download", "--server=127.0.0.1", "--port=30218", "--user=username", "--verbose",
		"--folder=_ALL_", "--path", maildir,
	}
	stdouterr := catchStdoutStderr(t)
	execute := setUpFakeServerAndCommand(t, args)

	err := execute()

	assert.NoError(t, err)
	_, stderr := stdouterr()

	// Ensure that the maildir looks as expected.
	actualFiles, actualDirs := scanDirReplacingEmails(t, maildir)

	expectedFiles := []string{
		".go-imapgrab.lock", "INBOX/new/email.0", "oldmail-127.0.0.1-30218-username-INBOX",
	}
	assert.Equal(t, expectedFiles, actualFiles)

	expectedDirs := []string{"", "INBOX", "INBOX/cur", "INBOX/new", "INBOX/tmp"}
	assert.Equal(t, expectedDirs, actualDirs)

	for _, msg := range defaultExpectedTestLogs {
		assert.Contains(t, stderr, msg)
	}
	for _, msg := range defaultExpectedDownloadTestLogs {
		assert.Contains(t, stderr, msg)
	}
}

func TestSystemLoginSuccess(t *testing.T) {
	args := []string{"login", "--server=127.0.0.1", "--port=30218", "--user=username", "--verbose"}

	stdouterr := catchStdoutStderr(t)
	execute := setUpFakeServerAndCommand(t, args)

	err := execute()
	require.NoError(t, err)

	stdout, stderr := stdouterr()

	assert.Contains(t, stdout, "Please provide your password for the following service:")
	assert.Contains(t, stdout, "Username: username")
	assert.Contains(t, stdout, "Server: 127.0.0.1")
	assert.Contains(t, stdout, "Port: 30218")
	assert.Empty(t, stderr)
}
