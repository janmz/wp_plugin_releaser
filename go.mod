module github.com/janmz/wp_plugin_release

go 1.25.0

require (
	github.com/janmz/sconfig v0.0.0
	github.com/janmz/ssh-commands v0.0.0
	github.com/nicksnyder/go-i18n/v2 v2.6.0
	golang.org/x/crypto v0.48.0
	golang.org/x/text v0.34.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	github.com/pkg/sftp v1.13.10 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/janmz/sconfig => ../sconfig

replace github.com/janmz/ssh-commands => ../ssh-commands
