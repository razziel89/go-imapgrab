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
	"time"

	"github.com/icza/gowut/gwu"
)

const (
	uiCellPadding  = 3
	uiNummailboxes = 10
	// Introductory text shown in the UI.
	uiIntroduction = "This is a simple UI for go-imapgrab.\n\nEnter details for new/updated " +
		"mailboxes in the text boxes at the top (ports must be integers, if in doubt, use 993 " +
		"for \"port\" and a unique number in [30100,30200] for \"serverport\"). " +
		"Separate folder specs by commas (use \"_ALL_, -_Gmail_\" for the \"folders\" box if in " +
		"doubt, leave out the quotes). " +
		"You need to save changes to mailbox configs using the \"save\" button to take effect. " +
		"Select which mailboxes to act upon in the list in the middle. Trigger actions " +
		"on all selected mailboxes with the buttons on the right. View logs at the very bottom. " +
		"If you download something, it may take quite a while until you see any changes. " +
		"The UI only refreshes once all actions have finished. " +
		"Initial downloads are particularly slow and may even result in a timeout. Any action " +
		"that does not finish within 1min will be cancelled automatically. The simplest fix is " +
		"to run the command that timed out in a terminal, the exact command is shown in the logs.\n"
	uiTimeout = time.Minute
)

// Contains all the components of the UI that are needed to provide functionality later on.
type uiElements struct {
	newMailboxDetailsTextboxes uiNewMailboxDetailsTextboxes
	actionButtons              uiActionButtons
	knownMailboxesList         gwu.ListBox
	verboseCheckbox            gwu.CheckBox
	reportLabel                gwu.Label
	window                     gwu.Window
}

type uiNewMailboxDetailsTextboxes struct {
	name       gwu.TextBox
	server     gwu.TextBox
	user       gwu.TextBox
	port       gwu.TextBox
	serverport gwu.TextBox
	password   gwu.TextBox
	folders    gwu.TextBox
}

type uiActionButtons struct {
	save     gwu.Button
	clear    gwu.Button
	login    gwu.Button
	list     gwu.Button
	download gwu.Button
	serve    gwu.Button
	edit     gwu.Button
	delete   gwu.Button
}

// Build the UI, excluding any and all functionality. This function is not tested separately because
// it simply constructs the UI. Any automated test of the UI arrangement would be complex and
// convoluted, hence none are added.
func uiBuild() uiElements {
	window := uiBuildMainWindow()
	newMailboxTextboxes, saveNewMailbox, clearBoxes, newMailboxPanel := uiBuildAddMailboxSection()
	knownMailboxesList, knownMailboxesPanel := uiBuildKnownMailboxesList()
	actionButtons, verboseCheckbox, actionButtonsPanel := uiBuildMailboxActionButtons()
	reportLabel := uiBuildReportLabel()

	// Make the action buttons part of the panel listing the mailboxes.
	knownMailboxesPanel.Add(actionButtonsPanel)

	// Add the save button separately.
	actionButtons.save = saveNewMailbox
	actionButtons.clear = clearBoxes

	// Add everything to the main window in the correct order.
	window.Add(newMailboxPanel)
	window.Add(knownMailboxesPanel)
	window.Add(reportLabel)

	return uiElements{
		window:                     window,
		reportLabel:                reportLabel,
		newMailboxDetailsTextboxes: newMailboxTextboxes,
		actionButtons:              actionButtons,
		knownMailboxesList:         knownMailboxesList,
		verboseCheckbox:            verboseCheckbox,
	}
}

func uiBuildMainWindow() gwu.Window {
	window := gwu.NewWindow("main", "go-imapgrab-ui")
	// Define some style elements for the window.
	window.Style().SetWidth("80%")

	// The introductory text, which is an integral part of the window. Without it, it would not be
	// clear how to use the UI.
	panel := gwu.NewVerticalPanel()
	panel.Style().SetWidth("80%")
	panel.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)
	panel.Add(gwu.NewLabel(uiIntroduction))
	window.Add(panel)

	return window
}

// Build text boxes to add a new mailbox entry, the button to trigger saving the thing, as well as
// the general panel containing that.
func uiBuildAddMailboxSection() (
	boxes uiNewMailboxDetailsTextboxes, saveButton, clearButton gwu.Button, panel gwu.Panel,
) {
	panel = gwu.NewVerticalPanel()
	panel.SetAlign(gwu.HARight, gwu.VADefault)
	panel.SetCellPadding(uiCellPadding)
	panel.Style().SetBorder2(1, gwu.BrdStyleSolid, gwu.ClrBlack)
	panel.Style().SetMargin("20px")
	panel.Add(gwu.NewLabel("Enter details for new/updated mailbox below:"))

	newBox := func(name string) gwu.TextBox {
		horPanel := gwu.NewHorizontalPanel()
		horPanel.SetAlign(gwu.HALeft, gwu.VAMiddle)
		label := gwu.NewLabel(name + ":")
		box := gwu.NewTextBox("")
		horPanel.Add(label)
		horPanel.Add(box)
		panel.Add(horPanel)
		return box
	}

	boxes = uiNewMailboxDetailsTextboxes{
		name:       newBox("Name"),
		server:     newBox("Server"),
		user:       newBox("User"),
		port:       newBox("Port"),
		serverport: newBox("Serverport"),
		password:   newBox("Password"),
		folders:    newBox("Folders"),
	}

	horPanel := gwu.NewHorizontalPanel()
	horPanel.SetCellPadding(uiCellPadding)
	saveButton = gwu.NewButton("Save")
	horPanel.Add(saveButton)
	clearButton = gwu.NewButton("Clear")
	horPanel.Add(clearButton)
	panel.Add(horPanel)

	return boxes, saveButton, clearButton, panel
}

// List of known mailboxes where boxes to act upon can be selected.
func uiBuildKnownMailboxesList() (gwu.ListBox, gwu.Panel) {
	panel := gwu.NewHorizontalPanel()
	panel.SetCellPadding(uiCellPadding)
	panel.Style().SetBorder2(1, gwu.BrdStyleSolid, gwu.ClrBlack)
	panel.Style().SetMargin("20px")
	panel.Add(gwu.NewLabel("Mailboxes:"))

	listBox := gwu.NewListBox(nil)
	listBox.SetRows(uiNummailboxes)
	listBox.SetMulti(true)

	panel.Add(listBox)
	return listBox, panel
}

// Add buttons to act on selected mailboxes.
func uiBuildMailboxActionButtons() (
	buttons uiActionButtons, verbose gwu.CheckBox, panel gwu.Panel,
) {
	panel = gwu.NewVerticalPanel()
	panel.SetCellPadding(uiCellPadding)

	verbose = gwu.NewCheckBox("Verbose Logs")
	panel.Add(verbose)

	newButton := func(name string) gwu.Button {
		button := gwu.NewButton(name + " Selected")
		panel.Add(button)
		return button
	}

	buttons = uiActionButtons{
		login:    newButton("Login"),
		list:     newButton("List"),
		download: newButton("Download"),
		serve:    newButton("Serve"),
		edit:     newButton("Edit"),
		delete:   newButton("Delete"),
		// Will be set externally since it is not part of this panel. This is a bit hacky but I
		// wanted to combine all the buttons in one type.
		save:  nil,
		clear: nil,
	}

	return buttons, verbose, panel
}

func uiBuildReportLabel() gwu.Label {
	label := gwu.NewLabel("Logs will show up here.")
	label.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)
	return label
}
