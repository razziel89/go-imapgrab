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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/icza/gowut/gwu"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	shortUIHelp = "Interact with go-imapgrab via a browser-based UI."
	uiPort      = 8081
	// Introductory text shown in the UI.
	uiIntroduction = "This is a simple UI for go-imapgrab.\nEnter details for new mailboxes in " +
		"the text boxes at the top.\nSelect which mailboxes to act upon in the list in the " +
		"middle.\nTrigger actions on all selected mailboxes with the buttons on the right.\n" +
		"View logs at the very bottom.\nIf you want to delete an entry, edit the config file.\n" +
		"If you download something, it may take quite a while until you see any changes.\n" +
		"The UI only refreshes once all actions have finished.\n" +
		"Initial downloads are particularly slow and may even result in a timeout.\n"
)

func getUICmd(keyring keyringOps, ops coreOps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Long:  shortUIHelp + "\n\n" + typicalFlowHelp,
		Short: shortUIHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find and parse the config file.
			cfgFile := findConfigFile()

			var uiConf uiConf
			if exists(cfgFile) {
				log.Printf("Using config file at %s", cfgFile)
				cfgContent, err := os.ReadFile(cfgFile) //nolint:gosec
				if err == nil {
					err = yaml.Unmarshal(cfgContent, &uiConf)
				}
				if err == nil {
					for _, mb := range uiConf.Mailboxes {
						// Ignore errors, e.g. because some credentials could not be found. This is
						// not optimal but at this point we just want to get the UI started and all
						// available passwords loaded.
						password, _ := retrieveFromKeyring(mb.asRootConf(), keyring)
						mb.password = password
					}
				}
				if err == nil && len(uiConf.Path) == 0 {
					err = fmt.Errorf(
						"empty path specified in config file %s, cannot download", cfgFile,
					)
				}
				if err != nil {
					return err
				}
			}

			// Run the UI.
			return runUI(&uiConf, cfgFile, ops, keyring, os.Args[0])
		},
	}
	return cmd
}

var uiCmd = getUICmd(defaultKeyring, &corer{})

func init() {
	rootCmd.AddCommand(uiCmd)
}

// Functionality to initialise the UI follows.

// Check whether a path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// Find the config file by searching some paths. By default, the file in
// XDG_CONFIG_HOME/go-imapgrab/config.yaml is being used. If that file cannot be found, try to use a
// file go-imapgrab.yaml in the current directory. If neither can be found, do not use a config file
// and simply start the UI.
func findConfigFile() string {
	xdgConfigHome, isSet := os.LookupEnv("XDG_CONFIG_HOME")
	if !isSet {
		xdgConfigHome = filepath.Join(os.Getenv("HOME"), ".config")
	}
	cfgInHome := filepath.Join(xdgConfigHome, "go-imapgrab", "config.yaml")
	if exists(cfgInHome) {
		return cfgInHome
	}
	cwd, err := os.Getwd()
	cfgInCDW := filepath.Join(cwd, "go-imapgrab.yaml")
	if err == nil && exists(cfgInCDW) {
		return cfgInCDW
	}
	_ = os.MkdirAll(filepath.Dir(cfgInHome), dirPerms)
	return cfgInHome
}

type uiConf struct {
	Path      string
	Mailboxes []*uiMailboxConf
}

type uiMailboxConf struct {
	Name       string
	Server     string
	User       string
	Port       int
	Serverport int
	// Keep this member internal so that it cannot be serialised or deserialised. It shall never be
	// written to a file but always retrieved from the keyring, if present.
	password string
}

func (mbCfg *uiMailboxConf) asRootConf() rootConfigT {
	return rootConfigT{
		server:    mbCfg.Server,
		port:      mbCfg.Port,
		username:  mbCfg.User,
		password:  mbCfg.password,
		verbose:   false,
		noKeyring: false,
	}
}

func (ui *uiConf) addMailbox(mailbox uiMailboxConf) {
	// Remove if already present. That means "adding" overwrites existing entries.
	existIdx := -1
	for idx, mb := range ui.Mailboxes {
		if mailbox.Name == mb.Name {
			existIdx = idx
		}
	}
	if existIdx >= 0 {
		// Replace an existing entry.
		mailboxes := append([]*uiMailboxConf{}, ui.Mailboxes[:existIdx]...)
		mailboxes = append(mailboxes, &mailbox)
		mailboxes = append(mailboxes, ui.Mailboxes[existIdx+1:]...)
		ui.Mailboxes = mailboxes
	} else {
		// Append a new entry.
		ui.Mailboxes = append(ui.Mailboxes, &mailbox)
	}
}

func (ui *uiConf) knownMailboxes() []string {
	result := make([]string, 0, len(ui.Mailboxes))
	for _, mb := range ui.Mailboxes {
		result = append(result, mb.Name)
	}
	return result
}

func (ui *uiConf) boxByName(name string) *uiMailboxConf {
	for _, mb := range ui.Mailboxes {
		if name == mb.Name {
			return mb
		}
	}
	return nil
}

const filePerms = 0644

func saveToFile(path string, cfg *uiConf, keyring keyringOps) error {
	fileContent, err := yaml.Marshal(cfg)
	if err == nil {
		err = os.WriteFile(path, fileContent, filePerms)
	}
	for _, mb := range cfg.Mailboxes {
		password, keyringErr := retrieveFromKeyring(mb.asRootConf(), keyring)
		if !credentialsNotFound(keyringErr) {
			err = errors.Join(err, keyringErr)
		}
		if err == nil && len(password) == 0 && len(mb.password) != 0 {
			// The password is not known but has been entered by the user, store it.
			keyringErr = addToKeyring(mb.asRootConf(), mb.password, keyring)
			err = errors.Join(err, keyringErr)
		}
	}
	if err != nil {
		err = fmt.Errorf("failed to save config: %s", err.Error())
	}
	return err
}

// UI specs follow.

type saveCfgEventHandler struct {
	cfg         *uiConf
	cfgPath     string
	boxes       map[string]gwu.TextBox
	reportLabel gwu.Label
	updates     []func(gwu.Event)
	keyring     keyringOps
}

func (h *saveCfgEventHandler) HandleEvent(event gwu.Event) {
	defer func() { event.MarkDirty(h.reportLabel) }()

	port, _ := strconv.Atoi(h.boxes["Port"].Text())
	serverport, _ := strconv.Atoi(h.boxes["Serverport"].Text())
	mb := uiMailboxConf{
		Name:       h.boxes["Name"].Text(),
		User:       h.boxes["User"].Text(),
		Server:     h.boxes["Server"].Text(),
		password:   h.boxes["Password"].Text(),
		Port:       port,
		Serverport: serverport,
	}

	if mb.Name == "" ||
		mb.User == "" ||
		mb.Server == "" ||
		mb.Port == 0 ||
		mb.Serverport == 0 ||
		mb.password == "" {

		h.reportLabel.SetText("Error in input values, at least\none value is unspecified!")
		h.reportLabel.Style().SetBackground(gwu.ClrRed)
		return
	}
	h.cfg.addMailbox(mb)
	if err := saveToFile(h.cfgPath, h.cfg, h.keyring); err != nil {
		h.reportLabel.SetText(err.Error())
		h.reportLabel.Style().SetBackground(gwu.ClrRed)
		return
	}
	h.reportLabel.SetText("Config successfully saved!")
	h.reportLabel.Style().SetBackground(gwu.ClrGreen)

	for _, box := range h.boxes {
		box.SetText("")
		event.MarkDirty(box)
	}

	// Update components that shall be refreshed.
	for _, update := range h.updates {
		update(event)
	}
}

const (
	uiCellPadding = 5
	contentSep    = "============================="
)

//nolint:funlen,gomnd
func runUI(cfg *uiConf, cfgPath string, _ coreOps, keyring keyringOps, pathToBin string) error {
	win := gwu.NewWindow("main", "go-imapgrab-ui")

	// Define some style elements.
	win.Style().SetWidth("80%")
	win.SetCellPadding(uiCellPadding)

	panel := gwu.NewVerticalPanel()
	panel.Style().SetWidth("80%")
	panel.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)
	panel.Add(gwu.NewLabel(uiIntroduction))
	win.Add(panel)

	// Text boxes to add a new entry.
	panel = gwu.NewVerticalPanel()
	panel.SetAlign(gwu.HARight, gwu.VADefault)
	panel.SetCellPadding(5)
	panel.Style().SetBorder2(1, gwu.BrdStyleSolid, gwu.ClrBlack)
	panel.Style().SetMargin("20px")
	panel.Add(gwu.NewLabel("Enter details for new mailbox below:"))
	boxes := map[string]gwu.TextBox{}
	for _, name := range []string{"Name", "Server", "User", "Port", "Serverport", "Password"} {
		horPanel := gwu.NewHorizontalPanel()
		horPanel.SetAlign(gwu.HALeft, gwu.VAMiddle)
		label := gwu.NewLabel(name + ":")
		box := gwu.NewTextBox("")
		box.AddSyncOnETypes(gwu.ETypeKeyUp)
		horPanel.Add(label)
		horPanel.Add(box)
		panel.Add(horPanel)
		boxes[name] = box
	}
	reportLabel := gwu.NewLabel("")
	reportLabel.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)
	btn := gwu.NewButton("Save")
	saveHandler := saveCfgEventHandler{
		cfg:         cfg,
		boxes:       boxes,
		reportLabel: reportLabel,
		cfgPath:     cfgPath,
		keyring:     keyring,
	}
	btn.AddEHandler(&saveHandler, gwu.ETypeClick)
	panel.Add(btn)
	panel.Add(reportLabel)
	win.Add(panel)

	// List of known mailboxes where boxes to act upon can be selected.
	panel = gwu.NewHorizontalPanel()
	panel.SetCellPadding(5)
	panel.Style().SetBorder2(1, gwu.BrdStyleSolid, gwu.ClrBlack)
	panel.Style().SetMargin("20px")
	panel.Add(gwu.NewLabel("All known mailboxes:"))
	// Define list and make sure it's updated when saving a new mailbox.
	listBox := gwu.NewListBox(nil)
	listBox.SetMulti(true)
	updateList := func(event gwu.Event) {
		listBox.SetRows(len(cfg.Mailboxes))
		listBox.SetValues(cfg.knownMailboxes())
		if event != nil {
			event.MarkDirty(listBox)
		}
	}
	updateList(nil)
	saveHandler.updates = append(saveHandler.updates, updateList)
	// Update an internal data structure that will always know which mailboxes are selected. That
	// way, we don't have to update it for every button that does something but we can just assume
	// it's there and up to date.
	selectedBoxes := []*uiMailboxConf{}
	listBox.AddEHandlerFunc(func(event gwu.Event) {
		newBoxes := []*uiMailboxConf{}
		for _, boxName := range listBox.SelectedValues() {
			if newBox := cfg.boxByName(boxName); newBox != nil {
				newBoxes = append(newBoxes, newBox)
			}
		}
		selectedBoxes = newBoxes
		log.Printf("selected: %v", selectedBoxes) // TODO: remove
		event.MarkDirty(listBox)
	}, gwu.ETypeChange)
	panel.Add(listBox)
	vertPanel := gwu.NewVerticalPanel()
	vertPanel.SetCellPadding(5)

	// Add buttons to act on selected mailboxes.
	reportLabel = gwu.NewLabel("")
	reportLabel.Style().SetWhiteSpace(gwu.WhiteSpacePreLine)

	verbose := gwu.NewCheckBox("Verbose Logs")
	vertPanel.Add(verbose)

	for _, buttonName := range []string{"Login", "List", "Download", "Serve"} {
		buttonName := buttonName
		button := gwu.NewButton(buttonName + " Selected")
		handler := func(event gwu.Event) {
			allContent := map[string]string{}
			wg := sync.WaitGroup{}
			wg.Add(len(selectedBoxes))
			for _, mb := range selectedBoxes {
				mb := mb
				go func() {
					content := []string{}
					args := []string{
						strings.ToLower(buttonName),
						// Ignore keyring, we are using env vars instead.
						"--no-keyring",
						"--server", mb.Server,
						"--user", mb.User,
						"--port", fmt.Sprint(mb.Port),
					}
					if verbose.State() {
						args = append(args, "--verbose")
					}
					stdin := ""
					if buttonName == "Serve" {
						args = append(args, []string{"--server-port", fmt.Sprint(mb.Serverport)}...)
						log.Fatal("cannot yet serve, don't know how to shut down", args)
					}
					if buttonName == "Download" {
						// Download all folders apart form Gmail-specific ones.
						args = append(args, []string{"--folder", "_ALL_"}...)
						args = append(args, []string{"--folder", "-_Gmail_"}...)
						args = append(args, []string{"--path", filepath.Join(cfg.Path, mb.Name)}...)
					}
					if buttonName == "Login" {
						stdin = mb.password
					}
					stdout, stderr, err := callWithArgs(
						pathToBin,
						args,
						[]string{fmt.Sprintf("%s=%s", passwdEnvVar, mb.password)},
						stdin,
					)
					if err != nil {
						content = append(
							content, fmt.Sprintf("Failure for %s, errors follow.\n", mb.Name),
						)
						content = append(content, err.Error())
					} else {
						content = append(content, fmt.Sprintf("Success for %s.\n", mb.Name))
					}
					if len(stdout) != 0 {
						content = append(content, fmt.Sprintf("Stdout for %s:\n", mb.Name))
						content = append(content, stdout)
					}
					if len(stderr) != 0 {
						content = append(content, fmt.Sprintf("Stderr for %s:\n", mb.Name))
						content = append(content, stderr)
					}
					content = append(content, contentSep)
					allContent[mb.Name] = strings.Join(content, "\n")
					log.Printf("Done processing %s", mb.Name)
					wg.Done()
				}()
			}
			wg.Wait()
			content := contentSep + "\n"
			for _, mb := range selectedBoxes {
				content += allContent[mb.Name] + "\n"
			}
			reportLabel.SetText(content)
			event.MarkDirty(reportLabel)
			log.Printf("Done doing %s", buttonName)
		}
		button.AddEHandlerFunc(handler, gwu.ETypeClick)
		vertPanel.Add(button)
	}

	panel.Add(vertPanel)
	win.Add(panel)
	win.Add(reportLabel)

	server := gwu.NewServer("go-imapgrab-ui", fmt.Sprintf("%s:%d", localhost, uiPort))
	server.SetText("go-imapgrab")
	err := server.AddWin(win)
	if err == nil {
		// Automatically connect to the main window. We do not want to support multiple windows.
		err = server.Start("main")
	}
	return err
}

// Call an executable with arguments and return stdout and stderr.
func callWithArgs(
	cmdName string,
	args []string,
	env []string,
	stdin string,
) (string, string, error) {
	log.Println("Running command:", cmdName, strings.Join(quote(args), " "))

	cmd := exec.Command(cmdName, args...)
	cmd.Env = env

	cmd.Stdin = strings.NewReader(stdin)

	stdout := strings.Builder{}
	cmd.Stdout = &stdout
	stderr := strings.Builder{}
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
