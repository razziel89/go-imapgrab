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
	"syscall"

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func getLoginCmd(
	rootConf *rootConfigT, keyring keyringOps, readPasswordFn func(int) ([]byte, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store credentials in your system's keyring.",
		RunE: func(cmd *cobra.Command, args []string) error {
			core.SetVerboseLogs(verbose)

			fmt.Printf(
				"Please provide your password for the following service:\n"+
					"  Username: %s\n  Server: %s\n  Port: %d\n\n"+
					"Your password won't be echoed. "+
					"You may need to reset your terminal after aborting with Ctrl+C.\n"+
					"\nPassword:",
				rootConf.username, rootConf.server, rootConf.port,
			)
			password, err := readPasswordFn(int(syscall.Stdin))
			if err == nil {
				err = addToKeyring(*rootConf, string(password), keyring)
			}
			return err
		},
	}

	return cmd
}

var loginCmd = getLoginCmd(&rootConf, defaultKeyring, term.ReadPassword)

func init() {
	initRootFlags(loginCmd, &rootConf)
	rootCmd.AddCommand(loginCmd)
}
