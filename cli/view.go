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

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

var viewConf viewConfigT

type viewConfigT struct {
	index  int
	folder string
}

func init() {
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a single email on your screen. Shows RFC822-compliant content.",
	RunE: func(cmd *cobra.Command, args []string) error {
		core.SetVerboseLogs(verbose)
		cfg := core.IMAPConfig{
			Server:   rootConf.server,
			Port:     rootConf.port,
			User:     rootConf.username,
			Password: rootConf.password,
		}
		email, err := core.PrintEmail(cfg, viewConf.folder, viewConf.index)
		if err != nil {
			return err
		}
		fmt.Println(email)
		return nil
	},
}

func init() {
	initViewFlags()
}

func initViewFlags() {
	pflags := viewCmd.PersistentFlags()

	pflags.StringVarP(&viewConf.folder, "folder", "f", "", "the folder to get an email from")
	pflags.IntVarP(&viewConf.index, "index", "i", 1, "index for email with 1 being most recent")
}
