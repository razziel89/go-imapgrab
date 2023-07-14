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

	"gopkg.in/yaml.v2"
)

type uiConfigFile struct {
	Path      string
	Mailboxes []*uiConfFileMailbox

	filePath string
}

type uiConfFileMailbox struct {
	Name       string
	Server     string
	User       string
	Port       int
	Serverport int
	// Keep this member internal so that it cannot be serialised or deserialised. It shall never be
	// written to a file but always retrieved from the keyring, if present.
	password string
}

func (mbCfg *uiConfFileMailbox) asRootConf() rootConfigT {
	return rootConfigT{
		server:    mbCfg.Server,
		port:      mbCfg.Port,
		username:  mbCfg.User,
		password:  mbCfg.password,
		verbose:   false,
		noKeyring: false,
	}
}

func (ui *uiConfigFile) upsertMailbox(mailbox uiConfFileMailbox) {
	// Remove if already present.
	existIdx := -1
	for idx, mb := range ui.Mailboxes {
		if mailbox.Name == mb.Name {
			existIdx = idx
		}
	}
	if existIdx >= 0 {
		// Replace an existing entry.
		mailboxes := append([]*uiConfFileMailbox{}, ui.Mailboxes[:existIdx]...)
		mailboxes = append(mailboxes, &mailbox)
		mailboxes = append(mailboxes, ui.Mailboxes[existIdx+1:]...)
		ui.Mailboxes = mailboxes
	} else {
		// Append a new entry.
		ui.Mailboxes = append(ui.Mailboxes, &mailbox)
	}
}

func (ui *uiConfigFile) knownMailboxes() []string {
	result := make([]string, 0, len(ui.Mailboxes))
	for _, mb := range ui.Mailboxes {
		result = append(result, mb.Name)
	}
	return result
}

func (ui *uiConfigFile) boxByName(name string) *uiConfFileMailbox {
	for _, mb := range ui.Mailboxes {
		if name == mb.Name {
			return mb
		}
	}
	return nil
}

func (ui *uiConfigFile) saveToFileAndKeyring(keyring keyringOps) error {
	// TODO: consider using a lock file when manipulating the config file.

	fileContent, err := yaml.Marshal(ui)
	if err == nil {
		err = os.WriteFile(ui.filePath, fileContent, filePerms)
	}
	if err == nil {
		for _, mb := range ui.Mailboxes {
			if err == nil && len(mb.password) != 0 {
				// The password has been entered by the user or it is known, store it. Note that that
				// means we will overwrite all existing passwords, too, but that is acceptablehere.
				// Saving the config is a rare event.
				err = addToKeyring(mb.asRootConf(), mb.password, keyring)
			}
		}
	}
	if err != nil {
		err = fmt.Errorf("failed to save config: %s", err.Error())
	}
	return err
}
