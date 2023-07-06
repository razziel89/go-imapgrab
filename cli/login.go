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
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"unicode"

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func quote(args []string) []string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		// Also quote the empty string.
		shallBeQuoted := arg == ""
		for _, char := range arg {
			if unicode.IsSpace(char) {
				shallBeQuoted = true
				break
			}
		}
		if shallBeQuoted {
			arg = fmt.Sprintf("\"%s\"", arg)
		}
		quoted = append(quoted, arg)
	}
	return quoted
}

func loginCmdUse(rootConf *rootConfigT, args []string) string {
	// Quote arguments that contain spaces.
	quoted := quote(args)

	// Construct an equivalent command line with only the command name replaced by "login".
	loginEquivalent := quote([]string{
		args[0], "login", "--server", rootConf.server, "--port", fmt.Sprint(rootConf.port),
		"--user", rootConf.username,
	})

	return fmt.Sprintf(
		"To store credentials in your system keyring, run\n\n  %s\n\n"+
			"Then enter your password at the prompt. Afterwards, run\n\n  %s\n\n"+
			"again and go-imapgrab will take the password from the keyring.\n",
		strings.Join(loginEquivalent, " "), strings.Join(quoted, " "),
	)
}

const shortLoginHelp = "Store credentials in your system's keyring."

type readPasswordFn func(int) ([]byte, error)

func getLoginCmd(
	rootConf *rootConfigT, keyring keyringOps, readPasswordFn readPasswordFn, ops coreOps,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Long:  shortLoginHelp + "\n\n" + typicalFlowHelp,
		Short: shortLoginHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			core.SetVerboseLogs(rootConf.verbose)
			// Allow insecure auth for local server for testing.
			insecure := rootConf.server == localhost
			cfg := core.IMAPConfig{
				Server:   rootConf.server,
				Port:     rootConf.port,
				User:     rootConf.username,
				Insecure: insecure,
				// Password will be filled in later.
				Password: "",
			}
			fmt.Printf(
				"Please provide your password for the following service:\n"+
					"  Username: %s\n  Server: %s\n  Port: %d\n\n"+
					"Your password won't be echoed as you type. "+
					"You may need to reset your terminal after aborting with Ctrl+C.\n"+
					"\nPassword:",
				cfg.User, cfg.Server, cfg.Port,
			)
			password, err := readPasswordFn(int(syscall.Stdin))
			cfg.Password = string(password)
			if err == nil {
				fmt.Printf(
					" PASSWORD NOT SHOWN\n\nTrying to connect to the IMAP server, please wait.\n\n",
				)
				err = ops.tryConnect(cfg)
			}
			var keyringErr error
			if err == nil && !rootConf.noKeyring {
				keyringErr = addToKeyring(*rootConf, cfg.Password, keyring)
			}
			if err == nil {
				var keyringMsg string
				if rootConf.noKeyring {
					keyringMsg = "not"
				} else if keyringErr != nil {
					log.Printf("ERROR addding password to keyring: %s\n", keyringErr.Error())
					keyringMsg = "could not be"
				} else {
					keyringMsg = "successfully"
				}
				fmt.Printf(
					"Credentials successfully validated. Password %s stored in keyring.\n",
					keyringMsg,
				)
			} else {
				fmt.Printf("\nCredentials could not be validated. Keyring unchanged.\n\n")
			}
			return err
		},
	}
	initRootFlags(cmd, rootConf)
	return cmd
}

func readFromStdin(fd int) ([]byte, error) {
	info, err := os.Stdin.Stat()
	if fd != 0 || err != nil || (info.Mode()&os.ModeCharDevice) == os.ModeCharDevice {
		// Input is a terminal or there was an error trying to determine that. Default to
		// interactive prompts.
		return term.ReadPassword(fd)
	}
	// Otherwise, make this work with pipes, too.
	return io.ReadAll(os.Stdin)
}

var loginCmd = getLoginCmd(&rootConfig, defaultKeyring, readFromStdin, &corer{})

func init() {
	rootCmd.AddCommand(loginCmd)
}
