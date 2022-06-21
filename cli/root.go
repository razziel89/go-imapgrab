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
	"log"

	"github.com/spf13/cobra"
)

const (
	defaultPort = 993
)

var (
	verbose  bool
	rootConf rootConfigT
	// Whether to disable use of the system keyring.
	noKeyring bool
)

type rootConfigT struct {
	server   string
	port     int
	username string
	password string
}

func logDebug(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func getRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "go-imapgrab",
		Short: "Backup your IMAP-based email accounts with ease.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

var rootCmd = getRootCmd()

func init() {
	initRootFlags(rootCmd)
}

func initRootFlags(rootCmd *cobra.Command) {
	pflags := rootCmd.PersistentFlags()

	pflags.StringVarP(&rootConf.server, "server", "s", "", "address of imap server")
	pflags.IntVarP(&rootConf.port, "port", "p", defaultPort, "login port for imap server")
	pflags.StringVarP(&rootConf.username, "user", "u", "", "login user name")
	pflags.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pflags.BoolVarP(&noKeyring, "no-keyring", "k", false, "do not use the system keyring")
}
