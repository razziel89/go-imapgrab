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
	"github.com/stretchr/testify/mock"
)

type mockCoreOps struct {
	mock.Mock
}

func (m *mockCoreOps) getAllFolders(cfg core.IMAPConfig) ([]string, error) {
	args := m.Called(cfg)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockCoreOps) downloadFolder(
	cfg core.IMAPConfig, folders []string, maildirBase string, threads int,
) error {
	args := m.Called(cfg, folders, maildirBase, threads)
	return args.Error(0)
}
