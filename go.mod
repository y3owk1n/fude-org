module org

go 1.27.0

require (
	github.com/y3owk1n/fude/api v0.3.0
	github.com/y3owk1n/fude/plugin v0.3.0
	github.com/y3owk1n/fude/sdk v0.3.0
)

require (
	github.com/tetratelabs/wazero v1.12.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

// The plugin needs fude's API after v0.3.0, so it builds against a checkout
// beside it until the next release.
replace (
	github.com/y3owk1n/fude/api => ../fude/api
	github.com/y3owk1n/fude/plugin => ../fude/plugin
	github.com/y3owk1n/fude/sdk => ../fude/sdk
)
