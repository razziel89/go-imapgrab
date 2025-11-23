module github.com/razziel89/go-imapgrab/cli

go 1.24.0

toolchain go1.24.10

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/emersion/go-imap v1.2.1
	github.com/razziel89/go-imapgrab/core v0.0.0-20250506185458-d3cd1a19519d
	github.com/rogpeppe/go-internal v1.14.1
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	github.com/zalando/go-keyring v0.2.6
	golang.org/x/term v0.37.0
)

require (
	al.essio.dev/pkg/shellescape v1.6.0 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/godbus/dbus/v5 v5.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
