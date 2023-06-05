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
	"path/filepath"
	"time"

	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

var serveConfig serveConfigT

type serveConfigT struct {
	path           string
	timeoutSeconds int
	serverPort     int
}

const shortServeHelp = "Serve a locally stored maildir backup to as an IMAP server."

func getServeCmd(
	rootConf *rootConfigT,
	serveConf *serveConfigT,
	keyring keyringOps,
	prodRun bool,
	ops coreOps,
	lockFn lockFn,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Long:  shortServeHelp + "\n\n" + typicalFlowHelp,
		Short: shortServeHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			core.SetVerboseLogs(verbose)
			// Allow insecure auth for local server for testing.
			insecure := rootConf.server == localhost
			cfg := core.IMAPConfig{
				Server:   rootConf.server,
				Port:     rootConf.port,
				User:     rootConf.username,
				Password: rootConf.password,
				Insecure: insecure,
			}
			lockfile := filepath.Join(serveConf.path, lockfileName)
			lockTimeout := time.Duration(serveConf.timeoutSeconds) * time.Second
			unlock, err := lockFn(lockfile, lockTimeout)
			if err != nil {
				return fmt.Errorf(
					"cannot get lock on local folder, another process might be using it: %s",
					err.Error(),
				)
			}
			defer unlock()
			return ops.serveMaildir(cfg, serveConf.serverPort, serveConf.path)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Do not use the keyring if it has been disabled globally or if this is a test run,
			// i.e. no prod run.
			disableKeyring := noKeyring || !prodRun
			err := initCredentials(rootConf, disableKeyring, keyring)
			if credentialsNotFound(err) {
				err = fmt.Errorf("%s\n\n%s", err.Error(), loginCmdUse(rootConf, os.Args))
			}
			return err
		},
	}
	initServeFlags(cmd, serveConf)
	initRootFlags(cmd, rootConf)
	return cmd
}

var serveCmd = getServeCmd(&rootConf, &serveConfig, defaultKeyring, true, &corer{}, lock)

func init() {
	rootCmd.AddCommand(serveCmd)
}

func initServeFlags(serveCmd *cobra.Command, serveConf *serveConfigT) {
	flags := serveCmd.Flags()

	flags.StringVar(&serveConf.path, "path", "", "the local path to your maildir's parent dir")
	flags.IntVar(
		&serveConf.serverPort, "server-port", 30912,
		"port on which the local IMAP server will listen",
	)
	flags.IntVar(
		&serveConf.timeoutSeconds, "timeout", defaultTimeoutSeconds,
		"time in seconds to wait for acquiring a lock on the local folder",
	)
}
