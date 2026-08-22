module github.com/bonarizki-dat/go-excel

// Minimum supported Go toolchain version. This is a deliberate
// requirement, not a claim that the library depends on 1.26 language
// features for performance; see README.md "When Not to Use This
// Library" for the rationale.
go 1.26

// Core dependencies - keep minimal for framework-agnostic design
require github.com/xuri/excelize/v2 v2.11.0

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/richardlehane/mscfb v1.0.7 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
