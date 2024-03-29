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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestListCommand(t *testing.T) {
	mockOps := mockCoreOps{}
	mockOps.On("getAllFolders", mock.Anything).Return([]string{}, fmt.Errorf("some error"))
	defer mockOps.AssertExpectations(t)

	t.Setenv("IGRAB_PASSWORD", "some password")

	mk := &mockKeyring{}

	rootConf := rootConfigT{}
	cmd := getListCmd(&rootConf, mk, &mockOps)
	rootConf.noKeyring = true

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestListCommandNoKeyringProdRun(t *testing.T) {
	mockOps := mockCoreOps{}
	// Nothing will be called because the keyring cannot be initialised and the password is not
	// given via an env var.
	defer mockOps.AssertExpectations(t)

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
	cmd := getListCmd(&rootConf, mk, &mockOps)
	rootConf.noKeyring = true

	err = cmd.Execute()
	assert.Error(t, err)
}

func TestListCommandNoCredentialsInKeyring(t *testing.T) {
	mockOps := mockCoreOps{}
	defer mockOps.AssertExpectations(t)

	user, err := user.Current()
	require.NoError(t, err)

	mk := &mockKeyring{}
	mk.On("Get", "go-imapgrab/@:993", user.Username).Return("", keyring.ErrNotFound)
	defer mk.AssertExpectations(t)

	rootConf := rootConfigT{}
	cmd := getListCmd(&rootConf, mk, &mockOps)

	err = cmd.Execute()
	assert.ErrorContains(t, err, "secret not found in keyring")
}
