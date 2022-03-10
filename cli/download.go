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
	"github.com/razziel89/go-imapgrab/core"
	"github.com/spf13/cobra"
)

var downloadConf downloadConfigT

type downloadConfigT struct {
	folders []string
	path    string
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download all not yet downloaded emails from a folder to a maildir.",
	RunE: func(cmd *cobra.Command, args []string) error {
		core.SetVerboseLogs(verbose)
		cfg := core.IMAPConfig{
			Server:   rootConf.server,
			Port:     rootConf.port,
			User:     rootConf.username,
			Password: rootConf.password,
		}
		err := core.DownloadFolder(cfg, downloadConf.folders, downloadConf.path)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	initDownloadFlags()
}

func initDownloadFlags() {
	pflags := downloadCmd.PersistentFlags()

	pflags.StringSliceVarP(
		&downloadConf.folders,
		"folder",
		"f",
		[]string{},
		"a folder spec specifying something to download (can be a folder name,\n"+
			"_ALL_ selects all folders, _Gmail_ selects Gmail folders, specify this\n"+
			"flag multiple times for multiple specs, prepend a minus '-' to any\n"+
			"spec to deselect instead, specs are interpreted in order)\n",
	)
	pflags.StringVar(&downloadConf.path, "path", "", "the local path to your maildir's parent dir")
}
