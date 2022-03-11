module github.com/razziel89/go-imapgrab/cli

go 1.16

replace github.com/razziel89/go-imapgrab/core => ../core

require (
	github.com/razziel89/go-imapgrab/core v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.3.0
	github.com/zalando/go-keyring v0.2.0
)
