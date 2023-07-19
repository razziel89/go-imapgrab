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
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setUpUIConfigFile(path string) uiConfigFile {
	return uiConfigFile{
		filePath: filepath.Join(path, "config.yaml"),
		Path:     filepath.Join(path, "download"),
		Mailboxes: []*uiConfFileMailbox{
			{
				Name:       "mail",
				Server:     "some.server.com",
				User:       "some@user.com",
				Port:       993,
				Serverport: 30123,
				Folders:    []string{"_ALL_", "-_Gmail_"},
				password:   "I am really secret",
			},
			{
				Name:       "box",
				Server:     "other.server.com",
				User:       "other@user.com",
				Port:       993,
				Serverport: 30124,
				Folders:    []string{"_ALL_"},
				password:   "I am very secret",
			},
		},
	}
}

// The yaml content that the above structure will be marshalled to.
func exampleConfigForTest(path string) string {
	return "" +
		"path: " + path + string(os.PathSeparator) + "download\n" +
		"mailboxes:\n" +
		"- name: mail\n" +
		"  server: some.server.com\n" +
		"  user: some@user.com\n" +
		"  port: 993\n" +
		"  serverport: 30123\n" +
		"  folders:\n" +
		"  - _ALL_\n" +
		"  - -_Gmail_\n" +
		"- name: box\n" +
		"  server: other.server.com\n" +
		"  user: other@user.com\n" +
		"  port: 993\n" +
		"  serverport: 30124\n" +
		"  folders:\n" +
		"  - _ALL_\n"
}

func TestBoxByName(t *testing.T) {
	cfg := setUpUIConfigFile(t.TempDir())
	assert.Nil(t, cfg.boxByName("unknown"))
	assert.NotNil(t, cfg.boxByName("box"))
}

func TestKnownMailboxes(t *testing.T) {
	cfg := setUpUIConfigFile(t.TempDir())
	assert.Equal(t, cfg.knownMailboxes(), []string{"mail", "box"})
}

func TestConvertedBoxByName(t *testing.T) {
	path := t.TempDir()

	root := &rootConfigT{
		server:    "other.server.com",
		port:      993,
		username:  "other@user.com",
		password:  "I am very secret",
		noKeyring: true,
		verbose:   false,
	}
	download := &downloadConfigT{
		path:           filepath.Join(path, "download", "box"),
		folders:        []string{"_ALL_"},
		threads:        0,
		timeoutSeconds: 1,
	}
	serve := &serveConfigT{
		path:           filepath.Join(path, "download", "box"),
		serverPort:     30124,
		timeoutSeconds: 1,
	}

	cfg := setUpUIConfigFile(path)

	assert.Equal(t, cfg.asRootConf("box", false), root)
	assert.Equal(t, cfg.asDownloadConf("box"), download)
	assert.Equal(t, cfg.asServeConf("box"), serve)

	assert.Nil(t, cfg.asRootConf("unknown", false))
	assert.Nil(t, cfg.asDownloadConf("unknown"))
	assert.Nil(t, cfg.asServeConf("unknown"))
}

func TestRemoveMailbox(t *testing.T) {
	path := t.TempDir()

	cfg := setUpUIConfigFile(path)

	assert.Equal(t, 2, len(cfg.Mailboxes))
	cfg.removeMailbox("unknown")
	assert.Equal(t, 2, len(cfg.Mailboxes))
	cfg.removeMailbox("mail")
	assert.Equal(t, 1, len(cfg.Mailboxes))
	cfg.removeMailbox("box")
	assert.Equal(t, 0, len(cfg.Mailboxes))
}

func TestUpsertMailbox(t *testing.T) {
	path := t.TempDir()

	cfg := setUpUIConfigFile(path)

	assert.Equal(t, 2, len(cfg.Mailboxes))

	box1 := uiConfFileMailbox{Name: "mail"}
	box2 := uiConfFileMailbox{Name: "box"}
	box3 := uiConfFileMailbox{Name: "mailbox"}

	cfg.upsertMailbox(box1)
	cfg.upsertMailbox(box2)
	cfg.upsertMailbox(box3)

	assert.Equal(t, cfg.Mailboxes, []*uiConfFileMailbox{&box1, &box2, &box3})
}

func TestSaveUIConfigToFile(t *testing.T) {
	user, err := user.Current()
	require.NoError(t, err)

	mockKeyring := mockKeyring{}
	mockKeyring.On(
		"Set", "go-imapgrab/some@user.com@some.server.com:993", user.Username, "I am really secret",
	).Return(nil)
	mockKeyring.On(
		"Set", "go-imapgrab/other@user.com@other.server.com:993", user.Username, "I am very secret",
	).Return(nil)

	defer mockKeyring.AssertExpectations(t)

	path := t.TempDir()
	cfg := setUpUIConfigFile(path)

	err = cfg.saveToFileAndKeyring(&mockKeyring)

	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(path, "config.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, string(content), exampleConfigForTest(path))
}

func TestFailedToSaveUIConfigToFile(t *testing.T) {
	user, err := user.Current()
	require.NoError(t, err)

	mockKeyring := mockKeyring{}
	mockKeyring.On(
		"Set", "go-imapgrab/some@user.com@some.server.com:993", user.Username, "I am really secret",
	).Return(fmt.Errorf("some error"))

	defer mockKeyring.AssertExpectations(t)

	path := t.TempDir()
	cfg := setUpUIConfigFile(path)

	err = cfg.saveToFileAndKeyring(&mockKeyring)

	assert.ErrorContains(t, err, "some error")

	content, err := os.ReadFile(filepath.Join(path, "config.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, string(content), exampleConfigForTest(path))
}
