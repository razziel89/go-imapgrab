module github.com/razziel89/go-imapgrab/cli

go 1.17

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/razziel89/go-imapgrab/core v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.3.0
	github.com/stretchr/testify v1.7.1
	github.com/zalando/go-keyring v0.2.0
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
)

require (
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/danieljoos/wincred v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-imap v1.2.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21 // indirect
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/text v0.3.8 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
