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
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/icza/gowut/gwu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockUIServer struct {
	mock.Mock
}

func (ms *mockUIServer) SetText(text string) {
	ms.Called(text)
}

func (ms *mockUIServer) AddWin(window gwu.Window) error {
	args := ms.Called(window)
	return args.Error(0)
}

func (ms *mockUIServer) Start(windows ...string) error {
	args := ms.Called(windows)
	return args.Error(0)
}

func TestFindUIConfigFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/asdf")

	path := findUIConfigFile()
	assert.Equal(t, path, "/asdf/.config/go-imapgrab/config.yaml")

	t.Setenv("XDG_CONFIG_HOME", "/blub")

	path = findUIConfigFile()
	assert.Equal(t, path, "/blub/go-imapgrab/config.yaml")

	// Ensure this test won't overwrite anything.
	require.False(t, exists("go-imapgrab.yaml"))

	// Otherwise, look in current directory.
	cwd, err := os.Getwd()
	require.NoError(t, err)
	err = os.WriteFile("go-imapgrab.yaml", nil, filePerms)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.Remove("go-imapgrab.yaml")) })

	path = findUIConfigFile()
	assert.Equal(t, path, filepath.Join(cwd, "go-imapgrab.yaml"))

	// If the file is present in home, return that one preferentially.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	configPath := filepath.Join(home, ".config", "go-imapgrab", "config.yaml")
	err = os.MkdirAll(filepath.Dir(configPath), dirPerms)
	require.NoError(t, err)
	err = os.WriteFile(configPath, nil, filePerms)
	require.NoError(t, err)

	path = findUIConfigFile()
	assert.Equal(t, path, configPath)
}

func TestGetUICommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// mockKeyring := mockKeyring{}
	// mockKeyring.On("Get", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
	//     Return("password", nil)
	//
	// defer mockKeyring.AssertExpectations(t)

	mockServer := mockUIServer{}
	mockServer.On("SetText", "go-imapgrab").Return()
	mockServer.On("AddWin", mock.Anything).Return(nil)
	mockServer.On("Start", []string{"main"}).Return(nil)

	defer mockServer.AssertExpectations(t)

	newServer := func(_ string, _ string) uiServer {
		return &mockServer
	}

	cmd := getUICmd(nil, newServer)
	err := cmd.Execute()

	assert.NoError(t, err)
}

func TestGetNewUI(t *testing.T) {
	cfgContent := "mailboxes: [{name: some-box, user: some-user}]"

	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte(cfgContent), filePerms))

	user, err := user.Current()
	require.NoError(t, err)

	mockKeyring := mockKeyring{}
	mockKeyring.On("Get", "go-imapgrab/some-user@:0", user.Username).Return("password", nil)

	defer mockKeyring.AssertExpectations(t)

	_, err = newUI(cfgFile, &mockKeyring)

	assert.NoError(t, err)
}

func TestRunUI(t *testing.T) {
	mockServer := mockUIServer{}
	mockServer.On("SetText", "go-imapgrab").Return()
	mockServer.On("AddWin", nil).Return(nil)
	mockServer.On("Start", []string{"main"}).Return(nil)

	defer mockServer.AssertExpectations(t)

	newServer := func(_ string, _ string) uiServer {
		return &mockServer
	}

	ui := &ui{}
	err := ui.run(newServer)

	assert.NoError(t, err)
}

func TestNewGwuServer(t *testing.T) {
	server := newGwuServer("some name", "localhost:30123")
	assert.NotNil(t, server)
}
