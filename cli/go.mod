module github.com/razziel89/go-imapgrab/cli

go 1.22.1

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/emersion/go-imap v1.2.1
	github.com/icza/gowut v1.4.0
	github.com/razziel89/go-imapgrab/core v0.0.0-20240728134458-371f821e8345
	github.com/rogpeppe/go-internal v1.13.1
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.10.0
	github.com/zalando/go-keyring v0.2.6
	golang.org/x/term v0.28.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-message v0.18.1 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
