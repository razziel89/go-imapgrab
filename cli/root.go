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
	"github.com/spf13/cobra"
)

const (
	defaultPort = 993
)

var rootConfig rootConfigT

type rootConfigT struct {
	server   string
	port     int
	username string
	password string
	verbose  bool
	// Whether to disable use of the system keyring.
	noKeyring bool
}

const (
	shortRootHelp   = "Back up your IMAP-based email accounts with ease."
	typicalFlowHelp = "" +
		"A typical run of go-imapgrab consists of 4 separate invocations. First, you store your\n" +
		"credentials in your system's keyring using the \"login\" command. Then, you list all\n" +
		"folders in your inbox using the \"list\" command. Next, you download all emails from\n" +
		"all folders that you wish to back up with the \"download\" command. For future runs in\n" +
		"in order to download only new emails, run the exact same \"download\" command again.\n" +
		"Last but not least, you open a local IMAP server using the \"serve\" command and use\n" +
		"your preferred email client to view your backed-up emails.\n\n" +
		"For an example Gmail account \"my.example@gmail.com\" with an application-specific\n" +
		"password \"example\", the typical call flow looks like this:\n\n" +
		"  go-imapgrab login --server imap.gmail.com --user my.example@gmail.com\n\n" +
		"  go-imapgrab list --server imap.gmail.com --user my.example@gmail.com\n\n" +
		"  go-imapgrab download --server imap.gmail.com --user my.example@gmail.com" +
		" --folder _ALL_ --folder -_Gmail_ --path backup\n\n" +
		"  go-imapgrab serve --server imap.gmail.com --user my.example@gmail.com" +
		" --path backup\n\n" +
		"Provide your password at the login prompt. Here, we download all folders apart from\n" +
		"Gmail-specific ones. Note that you may have to add the \"--port\" flag if your email\n" +
		"provider does not use the default of 993. The emails are downloaded to the local path\n" +
		"\"backup\". Connect to an IMAP server \"localhost\" on port 30912 with username\n" +
		"\"my.example@gmail.com\" and password \"example\" after running the \"serve\" command.\n"
)

func getRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go-imapgrab",
		Long:  shortRootHelp + "\n\n" + typicalFlowHelp,
		Short: shortRootHelp,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	return cmd
}

var rootCmd = getRootCmd()

func initRootFlags(rootCmd *cobra.Command, rootConf *rootConfigT) {
	flags := rootCmd.Flags()

	flags.StringVarP(&rootConf.server, "server", "s", "", "address of imap server")
	flags.IntVarP(&rootConf.port, "port", "p", defaultPort, "login port for imap server")
	flags.StringVarP(&rootConf.username, "user", "u", "", "login user name")
	flags.BoolVarP(&rootConf.verbose, "verbose", "v", false, "verbose output")
	flags.BoolVarP(&rootConf.noKeyring, "no-keyring", "k", false, "do not use the system keyring")
}
