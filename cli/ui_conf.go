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
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	Folders    []string
	// Keep this member internal so that it cannot be serialised or deserialised. It shall never be
	// written to a file but always retrieved from the keyring, if present.
	password string
}

func (mbCfg *uiConfFileMailbox) asRootConf(verbose bool) rootConfigT {
	return rootConfigT{
		server:   mbCfg.Server,
		port:     mbCfg.Port,
		username: mbCfg.User,
		password: mbCfg.password,
		verbose:  verbose,
		// Never use the keyring this way.
		noKeyring: true,
	}
}

func (mbCfg *uiConfFileMailbox) asDownloadConf(rootPath string) downloadConfigT {
	return downloadConfigT{
		folders:        mbCfg.Folders,
		path:           filepath.Join(rootPath, mbCfg.Name),
		threads:        0,
		timeoutSeconds: defaultTimeoutSeconds,
	}
}

func (mbCfg *uiConfFileMailbox) asServeConf(rootPath string) serveConfigT {
	return serveConfigT{
		path:           filepath.Join(rootPath, mbCfg.Name),
		serverPort:     mbCfg.Serverport,
		timeoutSeconds: defaultTimeoutSeconds,
	}
}

func (ui *uiConfigFile) removeMailbox(name string) {
	existIdx := -1
	for idx, mb := range ui.Mailboxes {
		if name == mb.Name {
			existIdx = idx
		}
	}
	if existIdx >= 0 {
		// Remove an existing entry.
		mailboxes := append([]*uiConfFileMailbox{}, ui.Mailboxes[:existIdx]...)
		mailboxes = append(mailboxes, ui.Mailboxes[existIdx+1:]...)
		ui.Mailboxes = mailboxes
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

	// The file will always be saved, even if the password cannot be stored in the keyring. That
	// gives the user the chance to try again later / after fixing keyring-related problems without
	// losing access to any provided information.
	fileContent, err := yaml.Marshal(ui)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(ui.filePath), dirPerms)
	}
	if err == nil {
		err = os.WriteFile(ui.filePath, fileContent, filePerms)
	}
	if err == nil {
		for _, mb := range ui.Mailboxes {
			if err == nil && len(mb.password) != 0 {
				// The password has been entered by the user or it is known, store it. Note that
				// that means we will overwrite all existing passwords, too, but that is
				// acceptablehere. Saving the config is a rare event.
				err = addToKeyring(mb.asRootConf(false), mb.password, keyring)
			}
		}
	}
	if err != nil {
		err = fmt.Errorf("failed to save config: %s", err.Error())
	}
	return err
}

func (ui *uiConfigFile) asRootConf(mailboxName string, verbose bool) *rootConfigT {
	box := ui.boxByName(mailboxName)
	if box == nil {
		return nil
	}
	result := box.asRootConf(verbose)
	return &result
}

func (ui *uiConfigFile) asDownloadConf(mailboxName string) *downloadConfigT {
	box := ui.boxByName(mailboxName)
	if box == nil {
		return nil
	}
	result := box.asDownloadConf(ui.Path)
	return &result
}

func (ui *uiConfigFile) asServeConf(mailboxName string) *serveConfigT {
	box := ui.boxByName(mailboxName)
	if box == nil {
		return nil
	}
	result := box.asServeConf(ui.Path)
	return &result
}
