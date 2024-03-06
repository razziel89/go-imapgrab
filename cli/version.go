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

	"github.com/spf13/cobra"
)

const devVersionString = "local-development"

var versionString string

func getVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of this executable.",
		RunE: func(_ *cobra.Command, _ []string) error {
			version := versionString
			if len(version) == 0 {
				version = devVersionString
			}
			fmt.Printf("version: %s", version)
			return nil
		},
	}

	return cmd
}

var versionCmd = getVersionCmd()

func init() {
	rootCmd.AddCommand(versionCmd)
}
