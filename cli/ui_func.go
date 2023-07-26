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
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/icza/gowut/gwu"
)

const (
	contentSep  = "============================="
	filePerms   = 0644
	uiTimeout   = 1 * time.Minute
	actionServe = "serve"
)

func uiFunctionalise(ui *ui) error {
	// Pre-populate elements.
	ui.elements.knownMailboxesList.SetValues(ui.config.knownMailboxes())

	reportFn := getUIReportFn(ui.elements.reportLabel)

	buttons := ui.elements.actionButtons
	uiAddButtonHandler(buttons.save, reportFn, ui, uiHandlerSave)
	uiAddButtonHandler(buttons.clear, reportFn, ui, uiHandlerClear)
	uiAddButtonHandler(buttons.login, reportFn, ui, uiHandlerLogin)
	uiAddButtonHandler(buttons.list, reportFn, ui, uiHandlerList)
	uiAddButtonHandler(buttons.download, reportFn, ui, uiHandlerDownload)
	uiAddButtonHandler(buttons.edit, reportFn, ui, uiHandlerEdit)
	uiAddButtonHandler(buttons.delete, reportFn, ui, uiHandlerDelete)
	uiAddButtonHandler(buttons.serve, reportFn, ui, uiHandlerServe)

	return nil
}

var (
	uiHandlerLogin    = getGenericUIButtonHandler("login", uiTimeout, runExeAsync)
	uiHandlerList     = getGenericUIButtonHandler("list", uiTimeout, runExeAsync)
	uiHandlerDownload = getGenericUIButtonHandler("download", uiTimeout, runExeAsync)
	uiHandlerServe    = getUIHandlerServe(runExeAsync)
)

type reportFn func(gwu.Event, string, string, error)

// Create a function that can be used to easily show output to the user.
func getUIReportFn(label gwu.Label) reportFn {
	return func(event gwu.Event, action, str string, err error) {
		var text string
		if err != nil {
			text = fmt.Sprintf("ERROR(S) executing action '%s':\n%s\n\n", action, err.Error())
		}
		text += str

		label.SetText(text)
		event.MarkDirty(label)
	}
}

type requestUpdateFn func(gwu.Comp)

type uiButtonHandlerFn func(*ui, requestUpdateFn) (string, error)

func uiAddButtonHandler(
	button gwu.Button, report reportFn, ui *ui, handler uiButtonHandlerFn,
) func(gwu.Event) {
	handlerFn := func(event gwu.Event) {
		// Make sure that no two handlers will ever be called at the same time.
		ui.mutex.Lock()
		defer ui.mutex.Unlock()

		str, err := handler(ui, func(comp gwu.Comp) { event.MarkDirty(comp) })
		report(event, button.Text(), str, err)
	}
	button.AddEHandlerFunc(handlerFn, gwu.ETypeClick)
	return handlerFn
}

// Handler functions follow.

func uiHandlerSave(ui *ui, update requestUpdateFn) (msg string, err error) {
	boxes := ui.elements.newMailboxDetailsTextboxes
	list := ui.elements.knownMailboxesList

	folders := []string{}
	for _, folder := range strings.Split(boxes.folders.Text(), ",") {
		folder := strings.TrimSpace(folder)
		if folder != "" {
			folders = append(folders, folder)
		}
	}

	port, _ := strconv.Atoi(strings.TrimSpace(boxes.port.Text()))
	serverport, _ := strconv.Atoi(strings.TrimSpace(boxes.serverport.Text()))
	mb := uiConfFileMailbox{
		Name:       strings.TrimSpace(boxes.name.Text()),
		User:       strings.TrimSpace(boxes.user.Text()),
		Server:     strings.TrimSpace(boxes.server.Text()),
		password:   strings.TrimSpace(boxes.password.Text()),
		Port:       port,
		Serverport: serverport,
		Folders:    folders,
	}

	if mb.Name == "" ||
		mb.User == "" ||
		mb.Server == "" ||
		mb.Port == 0 ||
		mb.Serverport == 0 ||
		mb.password == "" ||
		len(mb.Folders) == 0 {

		err = fmt.Errorf(
			"error in input values, at least one value is empty or zero or should be numeric " +
				"but is non-numeric",
		)
	}

	if err == nil {
		ui.config.upsertMailbox(mb)
		err = ui.config.saveToFileAndKeyring(ui.keyring)
	}

	if err == nil {
		msg = "Mailbox successfully saved!"
		// Request refreshes for all components that were affeced by this handler.
		for _, box := range []gwu.TextBox{
			boxes.name, boxes.password, boxes.port, boxes.server,
			boxes.serverport, boxes.user, boxes.folders,
		} {
			box.SetText("")
			update(box)
		}
		list.SetValues(ui.config.knownMailboxes())
		update(list)
	}

	return msg, err
}

func uiHandlerClear(ui *ui, update requestUpdateFn) (string, error) {
	boxes := ui.elements.newMailboxDetailsTextboxes

	// Request refreshes for all components that were affeced by this handler.
	for _, box := range []gwu.TextBox{
		boxes.name, boxes.password, boxes.port, boxes.server,
		boxes.serverport, boxes.user, boxes.folders,
	} {
		box.SetText("")
		update(box)
	}

	return "Textboxes successfully cleared!", nil
}

func uiHandlerDelete(ui *ui, update requestUpdateFn) (msg string, err error) {
	list := ui.elements.knownMailboxesList

	for _, box := range list.SelectedValues() {
		ui.config.removeMailbox(box)
	}
	err = ui.config.saveToFileAndKeyring(ui.keyring)

	if err == nil {
		msg = "Mailbox successfully removed!"
		// Request refreshes for all components that were affeced by this handler.
		list.SetValues(ui.config.knownMailboxes())
		update(list)
	}

	return msg, err
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
	boxes.folders.SetText(strings.Join(mb.Folders, ", "))

	// Request refreshes for all components that were affeced by this handler.
	for _, box := range []gwu.TextBox{
		boxes.name, boxes.password, boxes.port, boxes.server,
		boxes.serverport, boxes.user, boxes.folders,
	} {
		update(box)
	}

	return "Mailbox data loaded successfully!", nil
}

func getGenericUIButtonHandler(
	actionName string, timeout time.Duration, runExeAsync runExeAsyncFn,
) uiButtonHandlerFn {
	return func(ui *ui, _ requestUpdateFn) (string, error) {
		selectedBoxes := ui.elements.knownMailboxesList.SelectedValues()

		errs := []error{}
		outputs := []string{contentSep}
		addFns := []func(){}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		for _, box := range selectedBoxes {
			// Avoid loop variable weirdness.
			box := box

			root := ui.config.asRootConf(box, ui.elements.verboseCheckbox.State())
			download := ui.config.asDownloadConf(box)
			serve := ui.config.asServeConf(box)
			if root == nil || download == nil || serve == nil {
				log.Printf("skipping %s for unknown mailbox %s", actionName, box)
				continue
			}

			args, err := newRunSelfConf(ui.selfExe, actionName, *root, *download, *serve)
			if err != nil {
				return "", fmt.Errorf(
					"internal error while preparing to call self: %s", err.Error(),
				)
			}
			outputFn := runExeAsync(ctx, args)

			addFn := func() {
				output, err := outputFn()
				outputs = append(
					outputs, fmt.Sprintf("Mailbox: %s\n%s\n%s", box, output, contentSep),
				)
				errs = append(errs, err)
				log.Printf("Done processing %s", box)
			}
			addFns = append(addFns, addFn)
		}

		for _, fn := range addFns {
			fn()
		}
		log.Printf("Done processing all: %s", actionName)

		var err error
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("command not completed, timeout of %s reached", timeout)
		}

		return strings.Join(outputs, "\n"), errors.Join(err, errors.Join(errs...))
	}
}

func getUIHandlerServe(runExeAsync runExeAsyncFn) uiButtonHandlerFn {
	// The below function closes over these variables, which lets us avoid globals.
	var outputFns []func() (string, error)
	var ctx context.Context
	var cancel context.CancelFunc
	// At the beginning, assume we will be serving. Shut down processes if false.
	doServe := true

	return func(ui *ui, update requestUpdateFn) (string, error) {
		defer update(ui.elements.actionButtons.serve)
		if !doServe {
			cancel()
			for _, fn := range outputFns {
				// Always ignore returns here as it will only show that the binary was killed.
				_, _ = fn()
			}
			ui.elements.actionButtons.serve.SetText("Serve Selected")
			ui.elements.actionButtons.serve.Style().SetColor(gwu.ClrBlack)
			doServe = true
			return "Stopped serving.", nil
		}

		// Serve mailboxes. Initialise some variables.
		outputFns = []func() (string, error){}
		ctx, cancel = context.WithCancel(context.Background())
		what := []string{}

		for _, box := range ui.elements.knownMailboxesList.SelectedValues() {
			box := box

			root := ui.config.asRootConf(box, ui.elements.verboseCheckbox.State())
			download := ui.config.asDownloadConf(box)
			serve := ui.config.asServeConf(box)
			if root == nil || download == nil || serve == nil {
				log.Printf("skipping serve for unknown mailbox %s", box)
				continue
			}
			args, err := newRunSelfConf(ui.selfExe, actionServe, *root, *download, *serve)
			if err != nil {
				cancel()
				err = fmt.Errorf("internal error while preparing to call self: %s", err.Error())
				return "", err
			}

			outputFns = append(outputFns, runExeAsync(ctx, args))
			what = append(what, fmt.Sprintf("port %d: %s", serve.serverPort, box))
		}
		ui.elements.actionButtons.serve.SetText("Stop Serving")
		ui.elements.actionButtons.serve.Style().SetColor(gwu.ClrRed)
		doServe = false

		return fmt.Sprintf(
			"Serving %d mailboxes:\n%s",
			len(outputFns),
			strings.Join(what, "\n"),
		), nil
	}
}
