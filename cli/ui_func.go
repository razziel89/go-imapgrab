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

// func runUI(cfg *uiConf, cfgPath string, _ coreOps, keyring keyringOps, pathToBin string) error {
//     saveHandler := saveCfgEventHandler{
//         cfg:         cfg,
//         boxes:       boxes,
//         reportLabel: reportLabel,
//         cfgPath:     cfgPath,
//         keyring:     keyring,
//     }
//     btn.AddEHandler(&saveHandler, gwu.ETypeClick)
//     panel.Add(btn)
//     panel.Add(reportLabel)
//     win.Add(panel)
//
//     // List of known mailboxes where boxes to act upon can be selectedBoxes.
//     panel = gwu.NewHorizontalPanel()
//     panel.SetCellPadding(5)
//     panel.Style().SetBorder2(1, gwu.BrdStyleSolid, gwu.ClrBlack)
//     panel.Style().SetMargin("20px")
//     panel.Add(gwu.NewLabel("All known mailboxes:"))
//     // Define list and make sure it's updated when saving a new mailbox.
//     listBox := gwu.NewListBox(nil)
//     listBox.SetMulti(true)
//     updateList := func(event gwu.Event) {
//         listBox.SetRows(len(cfg.Mailboxes))
//         listBox.SetValues(cfg.knownMailboxes())
//         if event != nil {
//             event.MarkDirty(listBox)
//         }
//     }
//     updateList(nil)
//     saveHandler.updates = append(saveHandler.updates, updateList)
//     // Update an internal data structure that will always know which mailboxes are selectedBoxes. That
//     // way, we don't have to update it for every button that does something but we can just assume
//     // it's there and up to date.
//     selectedBoxes := []*uiMailboxConf{}
//     listBox.AddEHandlerFunc(func(event gwu.Event) {
//         newBoxes := []*uiMailboxConf{}
//         for _, boxName := range listBox.SelectedValues() {
//             if newBox := cfg.boxByName(boxName); newBox != nil {
//                 newBoxes = append(newBoxes, newBox)
//             }
//         }
//         selectedBoxes = newBoxes
//         log.Printf("selectedBoxes: %v", selectedBoxes) // TODO: remove
//         event.MarkDirty(listBox)
//     }, gwu.ETypeChange)
//     panel.Add(listBox)
//     vertPanel := gwu.NewVerticalPanel()
//     vertPanel.SetCellPadding(5)
//
//     // Add buttons to act on selectedBoxes mailboxes.
//     reportLabel = gwu.NewLabel("")
//     reportLabel.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)
//
//     verbose := gwu.NewCheckBox("Verbose Logs")
//     vertPanel.Add(verbose)
//
//     for _, buttonName := range []string{"Login", "List", "Download", "Serve"} {
//         buttonName := buttonName
//         button := gwu.NewButton(buttonName + " selectedBoxes")
//         handler := func(event gwu.Event) {
//             allContent := map[string]string{}
//             wg := sync.WaitGroup{}
//             wg.Add(len(selectedBoxes))
//             for _, mb := range selectedBoxes {
//                 mb := mb
//                 go func() {
//                     content := []string{}
//                     args := []string{
//                         strings.ToLower(buttonName),
//                         // Ignore keyring, we are using env vars instead.
//                         "--no-keyring",
//                         "--server", mb.Server,
//                         "--user", mb.User,
//                         "--port", fmt.Sprint(mb.Port),
//                     }
//                     if verbose.State() {
//                         args = append(args, "--verbose")
//                     }
//                     stdin := ""
//                     if buttonName == "Serve" {
//                         args = append(args, []string{"--server-port", fmt.Sprint(mb.Serverport)}...)
//                         log.Fatal("cannot yet serve, don't know how to shut down", args)
//                     }
//                     if buttonName == "Download" {
//                         // Download all folders apart form Gmail-specific ones.
//                         args = append(args, []string{"--folder", "_ALL_"}...)
//                         args = append(args, []string{"--folder", "-_Gmail_"}...)
//                         args = append(args, []string{"--path", filepath.Join(cfg.Path, mb.Name)}...)
//                     }
//                     if buttonName == "Login" {
//                         stdin = mb.password
//                     }
//                     stdout, stderr, err := callWithArgs(
//                         pathToBin,
//                         args,
//                         []string{fmt.Sprintf("%s=%s", passwdEnvVar, mb.password)},
//                         stdin,
//                     )
//                     if err != nil {
//                         content = append(
//                             content, fmt.Sprintf("Failure for %s, errors follow.\n", mb.Name),
//                         )
//                         content = append(content, err.Error())
//                     } else {
//                         content = append(content, fmt.Sprintf("Success for %s.\n", mb.Name))
//                     }
//                     if len(stdout) != 0 {
//                         content = append(content, fmt.Sprintf("Stdout for %s:\n", mb.Name))
//                         content = append(content, stdout)
//                     }
//                     if len(stderr) != 0 {
//                         content = append(content, fmt.Sprintf("Stderr for %s:\n", mb.Name))
//                         content = append(content, stderr)
//                     }
//                     content = append(content, contentSep)
//                     allContent[mb.Name] = strings.Join(content, "\n")
//                     log.Printf("Done processing %s", mb.Name)
//                     wg.Done()
//                 }()
//             }
//             wg.Wait()
//             content := contentSep + "\n"
//             for _, mb := range selectedBoxes {
//                 content += allContent[mb.Name] + "\n"
//             }
//             reportLabel.SetText(content)
//             event.MarkDirty(reportLabel)
//             log.Printf("Done doing %s", buttonName)
//         }
//         button.AddEHandlerFunc(handler, gwu.ETypeClick)
//         vertPanel.Add(button)
//     }
//
//     panel.Add(vertPanel)
//     win.Add(panel)
//     win.Add(reportLabel)
//
//     server := gwu.NewServer("go-imapgrab-ui", fmt.Sprintf("%s:%d", localhost, uiPort))
//     server.SetText("go-imapgrab")
//     err := server.AddWin(win)
//     if err == nil {
//         // Automatically connect to the main window. We do not want to support multiple windows.
//         err = server.Start("main")
//     }
//     return err
// }
