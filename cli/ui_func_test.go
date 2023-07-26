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
	"path/filepath"
	"sync"
	"testing"

	"github.com/icza/gowut/gwu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEvent struct {
	// Embed the original event interface to allow passing it in as a gwu.Event, even though most
	// methods are not implemented on it. That compensates for the inability to construct a
	// gwu.Event manually outside the gwu library. If a method is called that is not implemented, we
	// will get an access violation.
	gwu.Event
	mock.Mock
}

func (m *mockEvent) MarkDirty(comps ...gwu.Comp) {
	m.Called(comps)
}

func TestUIFunctionalise(t *testing.T) {
	ui := &ui{
		elements: uiBuild(),
		config:   uiConfigFile{},
		keyring:  nil,
		selfExe:  "cat",
	}

	buttons := ui.elements.actionButtons

	assertNumHanders := func(num int) {
		for _, button := range []gwu.Button{
			buttons.clear, buttons.delete, buttons.download, buttons.edit, buttons.list,
			buttons.login, buttons.save, buttons.serve,
		} {
			assert.Equal(t, num, button.HandlersCount(gwu.ETypeClick))
		}
	}
	assertNumHanders(0)

	err := uiFunctionalise(ui)

	assert.NoError(t, err)
	assertNumHanders(1)
}

func TestUIReportFn(t *testing.T) {
	label := gwu.NewLabel("some text")

	event := mockEvent{}
	event.On("MarkDirty", mock.Anything).Return()
	defer event.AssertExpectations(t)

	fn := getUIReportFn(label)

	fn(&event, "ACTION", "TEXT", nil)
	assert.Equal(t, "TEXT", label.Text())

	fn(&event, "ACTION", "TEXT", fmt.Errorf("SOME ERROR"))
	assert.Contains(t, label.Text(), "ERROR(S) executing action 'ACTION'")
	assert.Contains(t, label.Text(), "SOME ERROR")
	assert.Contains(t, label.Text(), "TEXT")
}

func TestUIAddButtonhandler(t *testing.T) {
	// Complex setup.
	reported := false
	report := func(_ gwu.Event, _ string, _ string, _ error) {
		reported = true
	}
	testUI := &ui{selfExe: "cat", mutex: sync.Mutex{}}
	button := gwu.NewButton("some button")
	handled := false
	innerHandler := func(_ *ui, report requestUpdateFn) (string, error) {
		report(nil)
		handled = true
		return "", nil
	}

	event := mockEvent{}
	event.On("MarkDirty", mock.Anything).Return()
	defer event.AssertExpectations(t)

	// Actual test.
	handler := uiAddButtonHandler(button, report, testUI, innerHandler)
	handler(&event)

	// Assertions.
	assert.True(t, reported)
	assert.True(t, handled)
}

func TestUIHandlerSave(t *testing.T) {
	// Setup.
	keyring := mockKeyring{}
	keyring.On("Set", mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("keyring error")).
		Once()
	keyring.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	tmp := t.TempDir()

	ui := &ui{
		elements: uiBuild(),
		config: uiConfigFile{
			filePath: filepath.Join(tmp, "config.yaml"),
			Path:     filepath.Join(tmp, "download"),
		},
		keyring: &keyring,
		selfExe: "cat",
	}

	updated := false
	update := func(_ gwu.Comp) { updated = true }

	// Test.
	_, err := uiHandlerSave(ui, update)

	// Assertions.
	assert.ErrorContains(t, err, "error in input values")
	assert.False(t, updated)
	keyring.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything)
	assert.False(t, exists(ui.config.filePath))

	// More setup.
	boxes := ui.elements.newMailboxDetailsTextboxes
	boxes.folders.SetText("_ALL_")
	boxes.name.SetText("name")
	boxes.password.SetText("password")
	boxes.port.SetText("1234")
	boxes.server.SetText("server")
	boxes.serverport.SetText("12345")
	boxes.user.SetText("user")

	// Test.
	_, err = uiHandlerSave(ui, update)

	// Assertions.
	assert.ErrorContains(t, err, "keyring error")
	assert.False(t, updated)
	keyring.AssertNumberOfCalls(t, "Set", 1)
	assert.True(t, exists(ui.config.filePath))

	// Test. Only the first attempt to save to the keyring causes an error. The second one succeeds.
	msg, err := uiHandlerSave(ui, update)

	// Assertions.
	assert.NoError(t, err)
	assert.True(t, updated)
	keyring.AssertNumberOfCalls(t, "Set", 2)
	assert.True(t, exists(ui.config.filePath))
	assert.NotEmpty(t, msg)
}
