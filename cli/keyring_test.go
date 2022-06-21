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
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockKeyring struct {
	mock.Mock
}

func (mk *mockKeyring) Get(service string, user string) (string, error) {
	args := mk.Called(service, user)
	return args.String(0), args.Error(1)
}

func (mk *mockKeyring) Set(service string, user string, password string) error {
	args := mk.Called(service, user, password)
	return args.Error(0)
}

func TestKeyringServiceSpec(t *testing.T) {
	cfg := rootConfigT{
		server:   "some server",
		port:     42,
		username: "some user",
		password: "will not be in spec",
	}

	spec := keyringServiceSpec(cfg)

	assert.NotContains(t, spec, "will not be in spec")
	assert.Contains(t, spec, "some server")
	assert.Contains(t, spec, "some user")
	assert.Contains(t, spec, "42")
}

func TestRetrieveFromKeyringSuccess(t *testing.T) {
	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i am not important",
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := mockKeyring{}
	mk.On("Get", "go-imapgrab/user@server:42", user.Username).Return("some password", nil)

	password, err := retrieveFromKeyring(cfg, &mk)

	assert.NoError(t, err)
	assert.Equal(t, "some password", password)
	mk.AssertExpectations(t)
}

func TestRetrieveFromKeyringNoPassword(t *testing.T) {
	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i am not important",
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := mockKeyring{}
	mk.On("Get", "go-imapgrab/user@server:42", user.Username).Return("", fmt.Errorf("some error"))

	password, err := retrieveFromKeyring(cfg, &mk)

	assert.Error(t, err)
	assert.Empty(t, password)
	mk.AssertExpectations(t)
}

func TestAddToKeyringSuccess(t *testing.T) {
	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := mockKeyring{}
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "i will be added").Return(nil)

	err = addToKeyring(cfg, cfg.password, &mk)

	assert.NoError(t, err)
	mk.AssertExpectations(t)
}

func TestAddToKeyringError(t *testing.T) {
	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}

	user, err := user.Current()
	assert.NoError(t, err)

	mk := mockKeyring{}
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "i will be added").
		Return(fmt.Errorf("some error"))

	err = addToKeyring(cfg, cfg.password, &mk)

	assert.Error(t, err)
	mk.AssertExpectations(t)
}

func TestInitCredentialsFromEnvironmentWithKeyring(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "some password")

	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}
	noKeyring = false

	user, err := user.Current()
	assert.NoError(t, err)

	mk := &mockKeyring{}
	// We expect the password to be auto-stored in the keyring in this case.
	mk.On("Set", "go-imapgrab/user@server:42", user.Username, "some password").
		Return(nil)

	err = initCredentials(&cfg, noKeyring, mk)

	assert.NoError(t, err)
	assert.Equal(t, cfg.password, "some password")
	mk.AssertExpectations(t)
}

func TestInitCredentialsFromEnvironmentNoKeyring(t *testing.T) {
	t.Setenv("IGRAB_PASSWORD", "some password")

	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}
	noKeyring = true
	mk := &mockKeyring{}

	err := initCredentials(&cfg, noKeyring, mk)

	assert.NoError(t, err)
	assert.Equal(t, cfg.password, "some password")
	// Make sure no keyring interaction took place.
	mk.AssertExpectations(t)
}

func TestInitCredentialsNoPasswordNoKeyring(t *testing.T) {
	if orgVal, found := os.LookupEnv("IGRAB_PASSWORD"); found {
		defer func() {
			err := os.Setenv("IGRAB_PASSWORD", orgVal)
			assert.NoError(t, err)
		}()
	}
	err := os.Unsetenv("IGRAB_PASSWORD")
	assert.NoError(t, err)

	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}
	noKeyring = true
	mk := &mockKeyring{}

	err = initCredentials(&cfg, noKeyring, mk)

	assert.Error(t, err)
	// Make sure the password has not been modified.
	assert.Equal(t, cfg.password, "i will be added")
	// Make sure no keyring interaction took place.
	mk.AssertExpectations(t)
}

func TestInitCredentialsNoPasswordFromKeyring(t *testing.T) {
	if orgVal, found := os.LookupEnv("IGRAB_PASSWORD"); found {
		defer func() {
			err := os.Setenv("IGRAB_PASSWORD", orgVal)
			assert.NoError(t, err)
		}()
	}
	err := os.Unsetenv("IGRAB_PASSWORD")
	assert.NoError(t, err)

	cfg := rootConfigT{
		server:   "server",
		port:     42,
		username: "user",
		password: "i will be added",
	}
	noKeyring = false

	user, err := user.Current()
	assert.NoError(t, err)

	mk := &mockKeyring{}
	mk.On("Get", "go-imapgrab/user@server:42", user.Username).
		Return("some password", nil)

	err = initCredentials(&cfg, noKeyring, mk)

	assert.NoError(t, err)
	assert.Equal(t, cfg.password, "some password")
	mk.AssertExpectations(t)
}

func TestDefaultKeyringGet(t *testing.T) {
	dk := defaultKeyringImpl{}
	_, _ = dk.Get("", "")
}

func TestDefaultKeyringSet(t *testing.T) {
	dk := defaultKeyringImpl{}
	_ = dk.Set("", "", "")
}
