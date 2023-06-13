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
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLoginSuccess(t *testing.T) {
	mockOps := mockCoreOps{}
	defer mockOps.AssertExpectations(t)
	mockOps.On("tryConnect", mock.Anything).Return(nil)

	rootConf := rootConfigT{password: "i do not matter"}
	calledReadPassword := false
	readPasswordFn := func(fd int) ([]byte, error) {
		// We read from stdin.
		assert.Equal(t, 0, fd)
		calledReadPassword = true
		return []byte("some password"), nil
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := &mockKeyring{}
	defer mk.AssertExpectations(t)
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "some password").Return(nil)

	cmd := getLoginCmd(&rootConf, mk, readPasswordFn, &mockOps)
	cmd.SetArgs([]string{"login", "--server=server", "--port=42", "--user=user"})
	err = cmd.Execute()

	assert.NoError(t, err)
	assert.True(t, calledReadPassword)
}

func TestLoginSuccessButKeyringError(t *testing.T) {
	mockOps := mockCoreOps{}
	defer mockOps.AssertExpectations(t)
	mockOps.On("tryConnect", mock.Anything).Return(nil)

	rootConf := rootConfigT{password: "i do not matter"}
	calledReadPassword := false
	readPasswordFn := func(fd int) ([]byte, error) {
		// We read from stdin.
		assert.Equal(t, 0, fd)
		calledReadPassword = true
		return []byte("some password"), nil
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := &mockKeyring{}
	defer mk.AssertExpectations(t)
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "some password").
		Return(fmt.Errorf("some keyring error"))

	cmd := getLoginCmd(&rootConf, mk, readPasswordFn, &mockOps)
	cmd.SetArgs([]string{"login", "--server=server", "--port=42", "--user=user"})
	err = cmd.Execute()

	assert.NoError(t, err)
	assert.True(t, calledReadPassword)
}

func TestLoginInterrupt(t *testing.T) {
	mockOps := mockCoreOps{}
	defer mockOps.AssertExpectations(t)

	rootConf := rootConfigT{}
	calledReadPassword := false
	readPasswordFn := func(fd int) ([]byte, error) {
		// We read from stdin.
		assert.Equal(t, 0, fd)
		calledReadPassword = true
		return []byte("some password"), fmt.Errorf("some error")
	}

	mk := &mockKeyring{}
	defer mk.AssertExpectations(t)

	cmd := getLoginCmd(&rootConf, mk, readPasswordFn, &mockOps)
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Equal(t, "some error", err.Error())
	assert.True(t, calledReadPassword)
}

func TestLoginCmdUseWithArgsWithSpaces(t *testing.T) {
	args := []string{
		"go-imapgrab", "command", "--flag", "arg w spaces", "--another-flag", "arg_wo_spaces",
	}
	rootConf := rootConfigT{
		server: "server w spaces", username: "username", port: 123, password: "not echoed",
	}

	helptext := loginCmdUse(&rootConf, args)

	assert.Contains(
		t, helptext, "go-imapgrab login --server \"server w spaces\" --port 123 --user username",
	)
	assert.Contains(
		t, helptext, "go-imapgrab command --flag \"arg w spaces\" --another-flag arg_wo_spaces",
	)
}
