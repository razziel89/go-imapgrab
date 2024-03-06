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
	"sort"
	"strings"

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

const shortListHelp = "Print all folders in your inbox."

func getListCmd(rootConf *rootConfigT, keyring keyringOps, ops coreOps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Long:  shortListHelp + "\n\n" + typicalFlowHelp,
		Short: shortListHelp,
		RunE: func(_ *cobra.Command, _ []string) error {
			core.SetVerboseLogs(rootConf.verbose)
			// Allow insecure auth for local server for testing.
			insecure := rootConf.server == localhost
			cfg := core.IMAPConfig{
				Server:   rootConf.server,
				Port:     rootConf.port,
				User:     rootConf.username,
				Password: rootConf.password,
				Insecure: insecure,
			}
			folders, err := ops.getAllFolders(cfg)

			sort.Strings(folders)
			fmt.Println(strings.Join(folders, "\n"))

			return err
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			core.SetVerboseLogs(rootConf.verbose)
			err := initCredentials(rootConf, keyring, rootConf.verbose)
			if credentialsNotFound(err) {
				err = fmt.Errorf("%s\n\n%s", err.Error(), loginCmdUse(rootConf, os.Args))
			}
			return err
		},
	}
	initRootFlags(cmd, rootConf)
	return cmd
}

var listCmd = getListCmd(&rootConfig, defaultKeyring, &corer{})

func init() {
	rootCmd.AddCommand(listCmd)
}
