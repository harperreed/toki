module github.com/harper/toki

go 1.24.9

// Use 2389-research fork of charm for self-hosted backend
replace github.com/charmbracelet/charm => github.com/2389-research/charm v0.0.0-20251219042514-41705ac0cc1c

require (
	github.com/fatih/color v1.18.0
	github.com/google/uuid v1.6.0
	github.com/modelcontextprotocol/go-sdk v1.1.0
	github.com/oklog/ulid/v2 v2.1.1
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	golang.org/x/term v0.38.0
	modernc.org/sqlite v1.40.1
)

require (
	github.com/harperreed/sweet v0.3.5
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/exp v0.0.0-20251125195548-87e1e737ad39 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.66.10 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)
