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
		ui.elements.reportLabel.Style().SetBackground(gwu.ClrWhite)

		var text string
		if err != nil {
			text = fmt.Sprintf("ERROR executing action '%s':\n%s\n\n", action, err.Error())
			// In case of errors, colour the background red to make that clear.
			ui.elements.reportLabel.Style().SetBackground(gwu.ClrRed)
		}
		text += str

		ui.elements.reportLabel.SetText(text)
		event.MarkDirty(ui.elements.reportLabel)
	}

	uiAddButtonHandler(ui.elements.actionButtons.save, reportFn, ui, uiHandlerSave)
	uiAddButtonHandler(ui.elements.actionButtons.list, reportFn, ui, uiHandlerList)

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

	return "Config successfully saved!", nil
}

func uiHandlerList(ui *ui, _ requestUpdateFn) (string, error) {
	selectedBoxes := ui.elements.knownMailboxesList.SelectedValues()

	errs := map[string]error{}
	outputs := map[string]string{}

	wg := sync.WaitGroup{}
	wg.Add(len(selectedBoxes))
	for _, box := range selectedBoxes {
		box := box
		go func() {
			output, err := runFromConf(
				ui.selfExe, "list",
				*ui.config.asRootConf(box, ui.elements.verboseCheckbox.State()),
				*ui.config.asDownloadConf(box),
				*ui.config.asServeConf(box),
			)
			outputs[box] = output
			errs[box] = err
			log.Printf("Done processing %s", box)
			wg.Done()
		}()
	}
	wg.Wait()
	log.Printf("Done processing all: list")

	var err error
	results := []string{contentSep}
	for _, box := range selectedBoxes {
		results = append(results, outputs[box])
		err = errors.Join(err, errs[box])
	}

	return strings.Join(results, "\n"), err
}
