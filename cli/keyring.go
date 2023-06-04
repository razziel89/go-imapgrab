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
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/zalando/go-keyring"
)

const (
	serviceName   = "go-imapgrab"
	serviceFormat = "%s/%s@%s:%d"
	passwdEnvVar  = "IGRAB_PASSWORD"
)

// Interface keyringOps abstracts away access to the keyring module.
type keyringOps interface {
	Get(service string, user string) (string, error)
	Set(service string, user string, password string) error
}

// Struct defaultKeyringImpl is the production implementation of the interface for the keyring
// module.
type defaultKeyringImpl struct{}

func (dk defaultKeyringImpl) Get(service string, user string) (string, error) {
	return keyring.Get(service, user)
}

func (dk defaultKeyringImpl) Set(service string, user string, password string) error {
	return keyring.Set(service, user, password)
}

var defaultKeyring keyringOps = &defaultKeyringImpl{}

// Function keyringServiceSpec provides a strig identifying a service with all its possible
// configuration components in the keyring.
func keyringServiceSpec(cfg rootConfigT) string {
	return fmt.Sprintf(serviceFormat, serviceName, cfg.username, cfg.server, cfg.port)
}

func retrieveFromKeyring(cfg rootConfigT, keyring keyringOps) (string, error) {
	serviceSpec := keyringServiceSpec(cfg)
	systemUserName, err := user.Current()

	var secret string
	if err == nil {
		secret, err = keyring.Get(serviceSpec, systemUserName.Username)
	}

	if err != nil {
		// Do not return anything that might have been retrieved in case of an error.
		secret = ""
	}

	return secret, err
}

func addToKeyring(cfg rootConfigT, password string, keyring keyringOps) error {
	serviceSpec := keyringServiceSpec(cfg)
	systemUserName, err := user.Current()

	if err == nil {
		err = keyring.Set(serviceSpec, systemUserName.Username, password)
	}

	return err
}

func initCredentials(rootConf *rootConfigT, noKeyring bool, keyring keyringOps) error {
	if password, found := os.LookupEnv(passwdEnvVar); found {
		logDebug(fmt.Sprintf("password taken from env var %s", passwdEnvVar))
		rootConf.password = password
		if noKeyring {
			return nil
		}
		logDebug("adding password to keyring")
		return addToKeyring(*rootConf, password, keyring)
	}

	if noKeyring {
		return fmt.Errorf("password not set via env var %s and keyring disabled", passwdEnvVar)
	}

	logDebug(fmt.Sprintf("password not set via env var %s, taking from keyring", passwdEnvVar))
	var err error
	rootConf.password, err = retrieveFromKeyring(*rootConf, keyring)
	return err
}

// Function credentialsNotFound determines whether the error you get when retrieving credentials
// indicates that the credentials could not be found.
func credentialsNotFound(err error) bool {
	return err != nil && errors.Is(err, keyring.ErrNotFound)
}
