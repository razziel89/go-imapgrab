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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestDownloadCommand(t *testing.T) {
	mockOps := mockCoreOps{}
	mockOps.On("downloadFolder", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("some error"))
	defer mockOps.AssertExpectations(t)

	lockCalled := false
	releaseCalled := false
	mockLock := func(_ string, _ time.Duration) (func(), error) {
		lockCalled = true
		return func() { releaseCalled = true }, nil
	}

	t.Setenv("IGRAB_PASSWORD", "some password")

	mk := &mockKeyring{}

	rootConfig.verbose = true

	rootConf := rootConfigT{}
	downloadConf := downloadConfigT{}
	cmd := getDownloadCmd(&rootConf, &downloadConf, mk, &mockOps, mockLock)

	// Overwrite rootConf since defaults are set by getDownloadCmd.
	rootConf.noKeyring = true

	err := cmd.Execute()
	assert.Error(t, err)
	assert.True(t, lockCalled)
	assert.True(t, releaseCalled)
}

func TestDownloadCommandNoKeyringProdRun(t *testing.T) {
	mockOps := mockCoreOps{}
	// Nothing will be called because the keyring cannot be initialised and the password is not
	// given via an env var.
	defer mockOps.AssertExpectations(t)

	lockCalled := false
	releaseCalled := false
	mockLock := func(_ string, _ time.Duration) (func(), error) {
		lockCalled = true
		return func() { releaseCalled = true }, nil
	}

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
	downloadConf := downloadConfigT{}
	cmd := getDownloadCmd(&rootConf, &downloadConf, mk, &mockOps, mockLock)

	// Overwrite rootConf since defaults are set by getDownloadCmd.
	rootConf.noKeyring = true

	err = cmd.Execute()
	assert.Error(t, err)
	// Lock functions not used at all in this error case.
	assert.False(t, lockCalled)
	assert.False(t, releaseCalled)
}

func TestDownloadCommandCannotGetLock(t *testing.T) {
	mockOps := mockCoreOps{}
	// Nothing will be called because the lock cannot be acqired.
	defer mockOps.AssertExpectations(t)

	lockCalled := false
	releaseCalled := false
	mockLock := func(_ string, _ time.Duration) (func(), error) {
		lockCalled = true
		return func() { releaseCalled = true }, fmt.Errorf("some locking error")
	}

	t.Setenv("IGRAB_PASSWORD", "some password")

	mk := &mockKeyring{}

	rootConf := rootConfigT{}
	downloadConf := downloadConfigT{}
	cmd := getDownloadCmd(&rootConf, &downloadConf, mk, &mockOps, mockLock)

	// Overwrite rootConf since defaults are set by getDownloadCmd.
	rootConf.noKeyring = true

	err := cmd.Execute()
	assert.Error(t, err)
	assert.True(t, lockCalled)
	// The release function is not called if we could not even obtain the lock.
	assert.False(t, releaseCalled)
}

func TestDownloadCommandNoCredentialsInKeyring(t *testing.T) {
	mockOps := mockCoreOps{}
	defer mockOps.AssertExpectations(t)

	mockLock := func(_ string, _ time.Duration) (func(), error) {
		t.Log("lock function should not be called")
		t.FailNow()
		return nil, fmt.Errorf("this should not be called")
	}

	user, err := user.Current()
	require.NoError(t, err)

	mk := &mockKeyring{}
	mk.On("Get", "go-imapgrab/@:993", user.Username).Return("", keyring.ErrNotFound)
	defer mk.AssertExpectations(t)

	cmd := getDownloadCmd(&rootConfigT{}, &downloadConfigT{}, mk, &mockOps, mockLock)

	err = cmd.Execute()
	assert.ErrorContains(t, err, "secret not found in keyring")
}
