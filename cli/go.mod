module github.com/razziel89/go-imapgrab/cli

go 1.22.1

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/emersion/go-imap v1.2.1
	github.com/icza/gowut v1.4.0
	github.com/razziel89/go-imapgrab/core v0.0.0-20240602201820-5f119e6aa367
	github.com/rogpeppe/go-internal v1.12.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/zalando/go-keyring v0.2.5
	golang.org/x/term v0.22.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/alessio/shellescape v1.4.2 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-message v0.18.1 // indirect
	github.com/emersion/go-sasl v0.0.0-20231106173351-e73c9f7bad43 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)
