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
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/icza/gowut/gwu"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	shortUIHelp = "Interact with go-imapgrab via a browser-based UI."
	uiPort      = 8081
)

func getUICmd(keyring keyringOps, newServer newServerFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Long:  shortUIHelp + "\n\n" + typicalFlowHelp,
		Short: shortUIHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgFile := findUIConfigFile()

			ui, err := newUI(cfgFile, keyring)
			if err == nil {
				err = uiFunctionalise(ui)
			}
			if err == nil {
				err = ui.run(newServer)
			}
			return err
		},
	}
	return cmd
}

// Create a wrapper function to enable the compiler to translate to the internal constructor type.
func newGwuServer(appName string, addr string) uiServer {
	return gwu.NewServer(appName, addr)
}

var uiCmd = getUICmd(defaultKeyring, newGwuServer)

func init() {
	rootCmd.AddCommand(uiCmd)
}

// Functionality apart form command specification above.

// Find the config file by searching some paths. By default, the file in
// XDG_CONFIG_HOME/go-imapgrab/config.yaml is being used. If that file cannot be found, try to use a
// file go-imapgrab.yaml in the current directory. If neither can be found, do not use a config file
// and simply start the UI.
func findUIConfigFile() string {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if len(xdgConfigHome) == 0 {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	cfgInHome := filepath.Join(xdgConfigHome, "go-imapgrab", "config.yaml")
	if exists(cfgInHome) {
		return cfgInHome
	}
	cwd, err := os.Getwd()
	cfgInCDW := filepath.Join(cwd, "go-imapgrab.yaml")
	if err == nil && exists(cfgInCDW) {
		return cfgInCDW
	}
	_ = os.MkdirAll(filepath.Dir(cfgInHome), dirPerms)
	return cfgInHome
}

type ui struct {
	elements uiElements
	config   uiConfigFile

	keyring keyringOps
	mutex   sync.Mutex
	selfExe string
}

func newUI(cfgFilePath string, keyring keyringOps) (*ui, error) {
	log.Printf("Using config file at %s", cfgFilePath)

	var uiConf uiConfigFile
	var cfgContent []byte
	var err error
	if exists(cfgFilePath) {
		cfgContent, err = os.ReadFile(cfgFilePath) //nolint:gosec
	}
	if err == nil {
		err = yaml.Unmarshal(cfgContent, &uiConf)
	}
	if err == nil {
		for _, mb := range uiConf.Mailboxes {
			// Ignore errors, e.g. because some credentials could not be found. This is not optimal
			// but at this point we just want to get the UI started and all available passwords
			// loaded.
			password, _ := retrieveFromKeyring(mb.asRootConf(false), keyring)
			mb.password = password
		}
	}
	if err == nil && len(uiConf.Path) == 0 {
		xdgState := os.Getenv("XDG_STATE_HOME")
		if len(xdgState) == 0 {
			xdgState = filepath.Join(os.Getenv("HOME"), ".local", "state")
		}
		uiConf.Path = filepath.Join(xdgState, "go-imapgrab", "download")
	}

	uiConf.filePath = cfgFilePath

	return &ui{
		elements: uiBuild(),
		config:   uiConf,
		keyring:  keyring,
		// Assume that the first value on os.Args is always the path to ourselves. That is almost
		// always true.
		selfExe: os.Args[0],
	}, err
}

type uiServer interface {
	SetText(string)
	AddWin(gwu.Window) error
	Start(...string) error
}

type newServerFn func(appName string, addr string) uiServer

func (ui *ui) run(newServer newServerFn) error {
	server := newServer("go-imapgrab-ui", fmt.Sprintf("%s:%d", localhost, uiPort))
	server.SetText("go-imapgrab")

	err := server.AddWin(ui.elements.window)
	if err == nil {
		// Automatically connect to the main window. We do not want to support multiple windows.
		err = server.Start("main")
	}
	return err
}
