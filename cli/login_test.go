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
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginSuccess(t *testing.T) {
	rootConf := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i do not matter",
	}
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
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "some password").Return(nil)

	cmd := getLoginCmd(&rootConf, mk, readPasswordFn)
	err = cmd.Execute()

	assert.NoError(t, err)
	assert.True(t, calledReadPassword)
}

func TestLoginInterrupt(t *testing.T) {
	rootConf := rootConfigT{}
	calledReadPassword := false
	readPasswordFn := func(fd int) ([]byte, error) {
		// We read from stdin.
		assert.Equal(t, 0, fd)
		calledReadPassword = true
		return []byte("some password"), fmt.Errorf("some error")
	}

	mk := &mockKeyring{}

	cmd := getLoginCmd(&rootConf, mk, readPasswordFn)
	err := cmd.Execute()

	assert.Error(t, err)
	assert.Equal(t, "some error", err.Error())
	assert.True(t, calledReadPassword)
}
