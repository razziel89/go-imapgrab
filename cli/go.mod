module github.com/razziel89/go-imapgrab/cli

go 1.27.0

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/emersion/go-imap v1.2.1
	github.com/razziel89/go-imapgrab/core v0.0.0-20260610184912-599ecb7a4ebc
	github.com/rogpeppe/go-internal v1.16.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.12.1
	github.com/zalando/go-keyring v0.2.8
	golang.org/x/term v0.46.0
)

require (
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
