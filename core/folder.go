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

package core

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// Constants used to expand folder name specs.
	allSelector     = "_ALL_"
	gmailSelector   = "_Gmail_"
	removalSelector = "-"
)

// All gmail-specific folders, identified via prefixes..
const (
	gmailPrefix1 = "[Gmail]"
	gmailPrefix2 = "[Google Mail]"
)

func isGmailDir(dirName string) bool {
	return strings.HasPrefix(dirName, gmailPrefix1) || strings.HasPrefix(dirName, gmailPrefix2)
}

// Perform fancy name replacements on folder names. For example, specifying _ALL_ causes all
// folders to be selected.
func expandFolders(folderSpecs, availableFolders []string) []string {
	logInfo(
		fmt.Sprintf("expanding folder spec '%s'", strings.Join(folderSpecs, logJoiner)),
	)
	logInfo(
		fmt.Sprintf("available folders are '%s'", strings.Join(availableFolders, logJoiner)),
	)
	// Convert to set to simplify manipulation.
	availableFoldersSet := setFromSlice(availableFolders)
	foldersSet := newOrderedSet(len(availableFolders))

	for _, folderSpec := range folderSpecs {
		if strings.HasPrefix(folderSpec, removalSelector) {
			folderSpec = strings.TrimPrefix(folderSpec, removalSelector)
			// Remove the specified directory.
			switch folderSpec {
			case allSelector:
				for _, removeMe := range availableFolders {
					foldersSet.remove(removeMe)
				}
			case gmailSelector:
				for _, removeMeCheck := range availableFolders {
					if isGmailDir(removeMeCheck) {
						foldersSet.remove(removeMeCheck)
					}
				}
			default:
				// Remove the specified folder, if it is known, log error otherwise.
				if !availableFoldersSet.has(folderSpec) {
					logError(fmt.Sprintf("ignoring attempted removal via spec %s", folderSpec))
				}
				foldersSet.remove(strings.TrimPrefix(folderSpec, removalSelector))
			}
		} else {
			// Add the specified directory.
			switch folderSpec {
			case allSelector:
				for _, addMe := range availableFolders {
					foldersSet.add(addMe)
				}
			case gmailSelector:
				for _, addMeCheck := range availableFolders {
					if isGmailDir(addMeCheck) {
						foldersSet.add(addMeCheck)
					}
				}
			default:
				foldersSet.add(folderSpec)
			}
		}
	}

	removed := foldersSet.exclusion(&availableFoldersSet).orderedEntries()
	warning := fmt.Sprintf("unselecting nonexisting folders '%s'", strings.Join(removed, logJoiner))
	if len(removed) > 0 {
		logWarning(warning)
	}
	folders := foldersSet.union(&availableFoldersSet).orderedEntries()
	sort.Strings(folders)
	logInfo(fmt.Sprintf("expanded to folders '%s'", strings.Join(folders, logJoiner)))
	return folders
}
