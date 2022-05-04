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

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
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
		folders, err := core.GetAllFolders(cfg)
		if err != nil {
			return err
		}

		sort.Strings(folders)

		for _, folder := range folders {
			fmt.Println(folder)
		}
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initCredentials()
	},
}
