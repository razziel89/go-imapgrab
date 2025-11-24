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

package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type serverMailbox struct {
	maildir  maildirPathT
	messages []*serverMessage
}

type pathAndInfo struct {
	path string
	info fs.FileInfo
}

func (mb *serverMailbox) addMessages() error {
	base := mb.maildir.folderPath()
	files := []pathAndInfo{}
	for _, dir := range []string{"new", "cur"} {
		moreFiles, err := os.ReadDir(filepath.Join(base, dir))
		if err != nil {
			return err
		}
		for idx := range moreFiles {
			// According to the docs of Info(), the only possible error is an ErrNotExists, which we
			// ignore here. We do not want to add a message that no longer exists on disk.
			info, err := moreFiles[idx].Info()
			if err == nil {
				files = append(files, pathAndInfo{
					path: filepath.Join(base, dir, moreFiles[idx].Name()),
					info: info,
				})
			}
		}
	}
	// Sort files by modification time to get some semblance of order.
	sort.Slice(files, func(i, j int) bool {
		return files[i].info.ModTime().Before(files[j].info.ModTime())
	})

	// Just store basic information, don't load full messages
	// Server functionality is minimal in v2 migration
	mb.messages = make([]*serverMessage, len(files))
	logInfo(fmt.Sprintf("read %d messags for mailbox %s", len(files), mb.maildir.folderName()))
	return nil
}
