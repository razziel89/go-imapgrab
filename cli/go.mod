module github.com/razziel89/go-imapgrab/cli

go 1.17

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/razziel89/go-imapgrab/core v0.0.0-20220804124858-8a7f7c93477d
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.7.1
	github.com/zalando/go-keyring v0.2.2
	golang.org/x/term v0.5.0
)

require (
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-imap v1.2.1 // indirect
	github.com/emersion/go-sasl v0.0.0-20220912192320-0145f2c60ead // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
