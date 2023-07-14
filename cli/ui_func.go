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
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/icza/gowut/gwu"
)

const (
	contentSep = "============================="
	filePerms  = 0644
)

func uiFunctionalise(ui *ui) error {
	// Pre-populate elements.
	ui.elements.knownMailboxesList.SetValues(ui.config.knownMailboxes())

	// Create a function that can be used to easily show output to the user.
	reportFn := func(event gwu.Event, action, str string, err error) {
		var text string
		if err != nil {
			text = fmt.Sprintf("ERROR(S) executing action '%s':\n%s\n\n", action, err.Error())
		}
		text += str

		ui.elements.reportLabel.SetText(text)
		event.MarkDirty(ui.elements.reportLabel)
	}

	buttons := ui.elements.actionButtons
	uiAddButtonHandler(buttons.save, reportFn, ui, uiHandlerSave)
	uiAddButtonHandler(buttons.login, reportFn, ui, getGenericUIButtonHandler("login"))
	uiAddButtonHandler(buttons.list, reportFn, ui, getGenericUIButtonHandler("list"))
	uiAddButtonHandler(buttons.download, reportFn, ui, getGenericUIButtonHandler("download"))
	uiAddButtonHandler(buttons.edit, reportFn, ui, uiHandlerEdit)
	uiAddButtonHandler(buttons.delete, reportFn, ui, uiHandlerDelete)

	return nil
}

type requestUpdateFn func(gwu.Comp)

type reportFn func(gwu.Event, string, string, error)

type uiButtomHandlerFn func(*ui, requestUpdateFn) (string, error)

func uiAddButtonHandler(
	button gwu.Button, report reportFn, ui *ui, handler uiButtomHandlerFn,
) {
	button.AddEHandlerFunc(
		func(event gwu.Event) {
			// Make sure that no two handlers will ever be called at the same time.
			ui.mutex.Lock()
			defer ui.mutex.Unlock()

			str, err := handler(ui, func(comp gwu.Comp) { event.MarkDirty(comp) })
			report(event, button.Text(), str, err)
		},
		gwu.ETypeClick,
	)
}

// Handler functions follow.

func uiHandlerSave(ui *ui, update requestUpdateFn) (string, error) {
	boxes := ui.elements.newMailboxDetailsTextboxes
	list := ui.elements.knownMailboxesList

	port, _ := strconv.Atoi(strings.TrimSpace(boxes.port.Text()))
	serverport, _ := strconv.Atoi(strings.TrimSpace(boxes.serverport.Text()))
	mb := uiConfFileMailbox{
		Name:       strings.TrimSpace(boxes.name.Text()),
		User:       strings.TrimSpace(boxes.user.Text()),
		Server:     strings.TrimSpace(boxes.server.Text()),
		password:   strings.TrimSpace(boxes.password.Text()),
		Port:       port,
		Serverport: serverport,
	}

	if mb.Name == "" ||
		mb.User == "" ||
		mb.Server == "" ||
		mb.Port == 0 ||
		mb.Serverport == 0 ||
		mb.password == "" {

		return "", fmt.Errorf("error in input values, at least one value is empty or zero")
	}
	ui.config.upsertMailbox(mb)
	if err := ui.config.saveToFileAndKeyring(ui.keyring); err != nil {
		return "", err
	}

	// Request refreshes for all components that were affeced by this handler.
	for _, box := range []gwu.TextBox{
		boxes.name, boxes.password, boxes.port, boxes.server, boxes.serverport, boxes.user,
	} {
		box.SetText("")
		update(box)
	}
	list.SetValues(ui.config.knownMailboxes())
	update(list)

	return "Mailbox successfully saved!", nil
}

func uiHandlerDelete(ui *ui, update requestUpdateFn) (string, error) {
	list := ui.elements.knownMailboxesList

	for _, box := range list.SelectedValues() {
		ui.config.removeMailbox(box)
	}
	if err := ui.config.saveToFileAndKeyring(ui.keyring); err != nil {
		return "", err
	}

	// Request refreshes for all components that were affeced by this handler.
	list.SetValues(ui.config.knownMailboxes())
	update(list)

	return "Mailbox successfully removed!", nil
}

func uiHandlerEdit(ui *ui, update requestUpdateFn) (string, error) {
	boxes := ui.elements.newMailboxDetailsTextboxes
	list := ui.elements.knownMailboxesList

	selected := list.SelectedValues()
	switch len(selected) {
	case 0:
		return "Select exactly one mailbox.", fmt.Errorf("too few mailboxes selected")
	case 1: // Success case, no-op.
	default:
		return "Select exactly one mailbox.", fmt.Errorf("too many mailboxes selected")
	}

	mb := ui.config.boxByName(selected[0])
	if mb == nil {
		return "", fmt.Errorf("internal error, selected mailbox is unknown")
	}

	boxes.name.SetText(mb.Name)
	boxes.password.SetText(mb.password)
	boxes.port.SetText(fmt.Sprint(mb.Port))
	boxes.server.SetText(mb.Server)
	boxes.serverport.SetText(fmt.Sprint(mb.Serverport))
	boxes.user.SetText(mb.User)

	// Request refreshes for all components that were affeced by this handler.
	for _, box := range []gwu.TextBox{
		boxes.name, boxes.password, boxes.port, boxes.server, boxes.serverport, boxes.user,
	} {
		update(box)
	}

	return "Mailbox data loaded successfully!", nil
}

func getGenericUIButtonHandler(actionName string) uiButtomHandlerFn {
	return func(ui *ui, _ requestUpdateFn) (string, error) {
		selectedBoxes := ui.elements.knownMailboxesList.SelectedValues()

		errs := map[string]error{}
		outputs := map[string]string{}

		wg := sync.WaitGroup{}
		wg.Add(len(selectedBoxes))
		for _, box := range selectedBoxes {
			box := box
			go func() {
				output, err := runFromConf(
					ui.selfExe, actionName,
					*ui.config.asRootConf(box, ui.elements.verboseCheckbox.State()),
					*ui.config.asDownloadConf(box),
					*ui.config.asServeConf(box),
				)
				outputs[box] = fmt.Sprintf("Mailbox: %s\n%s\n%s", box, output, contentSep)
				errs[box] = err
				log.Printf("Done processing %s", box)
				wg.Done()
			}()
		}
		wg.Wait()
		log.Printf("Done processing all: %s", actionName)

		var err error
		results := []string{contentSep}
		for _, box := range selectedBoxes {
			results = append(results, outputs[box])
			err = errors.Join(err, errs[box])
		}

		return strings.Join(results, "\n"), err
	}
}
