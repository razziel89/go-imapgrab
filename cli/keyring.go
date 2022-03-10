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

	"github.com/zalando/go-keyring"
)

const (
	serviceName   = "go-imapgrab"
	serviceFormat = "%s/%s@%s:%d"
)

func keyringServiceSpec(cfg rootConfigT) string {
	return fmt.Sprintf(serviceFormat, serviceName, cfg.username, cfg.server, cfg.port)
}

func retrieveFromKeyring(cfg rootConfigT) (string, error) {
	serviceSpec := keyringServiceSpec(cfg)
	systemUserName, err := user.Current()
	if err != nil {
		return "", err
	}

	secret, err := keyring.Get(serviceSpec, systemUserName.Username)
	if err != nil {
		return "", err
	}

	return secret, nil
}

func addToKeyring(cfg rootConfigT, password string) error {
	serviceSpec := keyringServiceSpec(cfg)
	systemUserName, err := user.Current()
	if err != nil {
		return err
	}

	err = keyring.Set(serviceSpec, systemUserName.Username, password)
	if err != nil {
		return err
	}

	return nil
}
