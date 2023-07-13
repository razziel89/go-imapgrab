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
	"log"
	"os"
	"path/filepath"

	"github.com/icza/gowut/gwu"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	shortUIHelp = "Interact with go-imapgrab via a browser-based UI."
	uiPort      = 8081
)

func getUICmd(keyring keyringOps, ops coreOps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Long:  shortUIHelp + "\n\n" + typicalFlowHelp,
		Short: shortUIHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find and parse the config file.
			cfgFile := findConfigFile()

			var uiConf uiConf
			if exists(cfgFile) {
				log.Printf("Using config file at %s", cfgFile)
				cfgContent, err := os.ReadFile(cfgFile) //nolint:gosec
				if err == nil {
					err = yaml.Unmarshal(cfgContent, &uiConf)
				}
				if err == nil {
					for _, mb := range uiConf.Mailboxes {
						// Ignore errors, e.g. because some credentials could not be found. This is
						// not optimal but at this point we just want to get the UI started and all
						// available passwords loaded.
						password, _ := retrieveFromKeyring(mb.asRootConf(), keyring)
						mb.password = password
					}
				}
				if err == nil && len(uiConf.Path) == 0 {
					err = fmt.Errorf(
						"empty path specified in config file %s, cannot download", cfgFile,
					)
				}
				if err != nil {
					return err
				}
			}

			// Run the UI.
			return runUI(&uiConf, cfgFile, ops, keyring, os.Args[0])
		},
	}
	return cmd
}

var uiCmd = getUICmd(defaultKeyring, &corer{})

func init() {
	rootCmd.AddCommand(uiCmd)
}

// Functionality to initialise the UI follows.

// Check whether a path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// Find the config file by searching some paths. By default, the file in
// XDG_CONFIG_HOME/go-imapgrab/config.yaml is being used. If that file cannot be found, try to use a
// file go-imapgrab.yaml in the current directory. If neither can be found, do not use a config file
// and simply start the UI.
func findConfigFile() string {
	xdgConfigHome, isSet := os.LookupEnv("XDG_CONFIG_HOME")
	if !isSet {
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

type uiConf struct {
	Path      string
	Mailboxes []*uiMailboxConf
}

type uiMailboxConf struct {
	Name       string
	Server     string
	User       string
	Port       int
	Serverport int
	// Keep this member internal so that it cannot be serialised or deserialised. It shall never be
	// written to a file but always retrieved from the keyring, if present.
	password string
}

func (mbCfg *uiMailboxConf) asRootConf() rootConfigT {
	return rootConfigT{
		server:    mbCfg.Server,
		port:      mbCfg.Port,
		username:  mbCfg.User,
		password:  mbCfg.password,
		verbose:   false,
		noKeyring: false,
	}
}

//
// func (ui *uiConf) addMailbox(mailbox uiMailboxConf) {
//     // Remove if already present. That means "adding" overwrites existing entries.
//     existIdx := -1
//     for idx, mb := range ui.Mailboxes {
//         if mailbox.Name == mb.Name {
//             existIdx = idx
//         }
//     }
//     if existIdx >= 0 {
//         // Replace an existing entry.
//         mailboxes := append([]*uiMailboxConf{}, ui.Mailboxes[:existIdx]...)
//         mailboxes = append(mailboxes, &mailbox)
//         mailboxes = append(mailboxes, ui.Mailboxes[existIdx+1:]...)
//         ui.Mailboxes = mailboxes
//     } else {
//         // Append a new entry.
//         ui.Mailboxes = append(ui.Mailboxes, &mailbox)
//     }
// }
//
// func (ui *uiConf) knownMailboxes() []string {
//     result := make([]string, 0, len(ui.Mailboxes))
//     for _, mb := range ui.Mailboxes {
//         result = append(result, mb.Name)
//     }
//     return result
// }
//
// func (ui *uiConf) boxByName(name string) *uiMailboxConf {
//     for _, mb := range ui.Mailboxes {
//         if name == mb.Name {
//             return mb
//         }
//     }
//     return nil
// }
//
// const filePerms = 0644
//
// func saveToFile(path string, cfg *uiConf, keyring keyringOps) error {
//     fileContent, err := yaml.Marshal(cfg)
//     if err == nil {
//         err = os.WriteFile(path, fileContent, filePerms)
//     }
//     for _, mb := range cfg.Mailboxes {
//         password, keyringErr := retrieveFromKeyring(mb.asRootConf(), keyring)
//         if !credentialsNotFound(keyringErr) {
//             err = errors.Join(err, keyringErr)
//         }
//         if err == nil && len(password) == 0 && len(mb.password) != 0 {
//             // The password is not known but has been entered by the user, store it.
//             keyringErr = addToKeyring(mb.asRootConf(), mb.password, keyring)
//             err = errors.Join(err, keyringErr)
//         }
//     }
//     if err != nil {
//         err = fmt.Errorf("failed to save config: %s", err.Error())
//     }
//     return err
// }
//
// // UI specs follow.
//
// type saveCfgEventHandler struct {
//     cfg         *uiConf
//     cfgPath     string
//     boxes       map[string]gwu.TextBox
//     reportLabel gwu.Label
//     updates     []func(gwu.Event)
//     keyring     keyringOps
// }
//
// func (h *saveCfgEventHandler) HandleEvent(event gwu.Event) {
//     defer func() { event.MarkDirty(h.reportLabel) }()
//
//     port, _ := strconv.Atoi(h.boxes["Port"].Text())
//     serverport, _ := strconv.Atoi(h.boxes["Serverport"].Text())
//     mb := uiMailboxConf{
//         Name:       h.boxes["Name"].Text(),
//         User:       h.boxes["User"].Text(),
//         Server:     h.boxes["Server"].Text(),
//         password:   h.boxes["Password"].Text(),
//         Port:       port,
//         Serverport: serverport,
//     }
//
//     if mb.Name == "" ||
//         mb.User == "" ||
//         mb.Server == "" ||
//         mb.Port == 0 ||
//         mb.Serverport == 0 ||
//         mb.password == "" {
//
//         h.reportLabel.SetText("Error in input values, at least\none value is unspecified!")
//         h.reportLabel.Style().SetBackground(gwu.ClrRed)
//         return
//     }
//     h.cfg.addMailbox(mb)
//     if err := saveToFile(h.cfgPath, h.cfg, h.keyring); err != nil {
//         h.reportLabel.SetText(err.Error())
//         h.reportLabel.Style().SetBackground(gwu.ClrRed)
//         return
//     }
//     h.reportLabel.SetText("Config successfully saved!")
//     h.reportLabel.Style().SetBackground(gwu.ClrGreen)
//
//     for _, box := range h.boxes {
//         box.SetText("")
//         event.MarkDirty(box)
//     }
//
//     // Update components that shall be refreshed.
//     for _, update := range h.updates {
//         update(event)
//     }
// }

func runUI(_ *uiConf, _ string, _ coreOps, _ keyringOps, _ string) error {
	ui := uiBuild()
	err := uiFunctionalise(&ui)

	var server gwu.Server
	if err == nil {
		server = gwu.NewServer("go-imapgrab-ui", fmt.Sprintf("%s:%d", localhost, uiPort))
		server.SetText("go-imapgrab")
		err = server.AddWin(ui.window)
	}
	if err == nil {
		// Automatically connect to the main window. We do not want to support multiple windows.
		err = server.Start("main")
	}
	return err
}
