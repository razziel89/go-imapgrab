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

package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultPort = 993
)

var (
	verbose  bool
	cfgFile  string
	rootConf generalConfigT
)

type generalConfigT struct {
	server   string
	port     int
	username string
	password string
}

var rootCmd = &cobra.Command{
	Use:   "imapgrab",
	Short: "Backup your IMAP-based email accounts with ease.",
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(cmd.Help())
	},
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	pflags := rootCmd.PersistentFlags()

	pflags.StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/imapgrab.yaml)")
	pflags.StringVarP(&rootConf.server, "server", "s", "", "address of imap server")
	pflags.IntVarP(&rootConf.port, "port", "p", defaultPort, "login port for imap server")
	pflags.StringVarP(&rootConf.username, "user", "u", "", "login user name")
	pflags.BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if password, found := os.LookupEnv("IGRAB_PASSWORD"); !found && verbose {
		log.Println("warning: password not set via env var IGRAB_PASSWORD")
	} else {
		rootConf.password = password
	}
}

func initConfig() {
	if len(cfgFile) != 0 {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search for default imapgrab config file.
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName("imapgrab")
	}

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			log.Println("using config file:", viper.ConfigFileUsed())
		}
	} else {
		if verbose {
			log.Println("not using config file")
		}
	}
}
