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
	"sort"
	"strings"

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

func getListCmd(rootConf *rootConfigT, keyring keyringOps, prodRun bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print all folders in your inbox.",
		RunE: func(cmd *cobra.Command, args []string) error {
			core.SetVerboseLogs(verbose)
			cfg := core.IMAPConfig{
				Server:   rootConf.server,
				Port:     rootConf.port,
				User:     rootConf.username,
				Password: rootConf.password,
			}
			imapgrabOps := core.NewImapgrabOps()
			folders, err := core.GetAllFolders(cfg, imapgrabOps)

			sort.Strings(folders)
			fmt.Println(strings.Join(folders, "\n"))

			return err
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Do not use the keyring if it has been disabled globally or if this is a test run,
			// i.e. no prod run.
			disableKeyring := noKeyring || !prodRun
			return initCredentials(rootConf, disableKeyring, keyring)
		},
	}

	return cmd
}

var listCmd = getListCmd(&rootConf, defaultKeyring, true)

func init() {
	rootCmd.AddCommand(listCmd)
}
