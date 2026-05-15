module github.com/oneliang/aura/mcp

go 1.26.1

require (
	github.com/mark3labs/mcp-go v0.47.1
	github.com/oneliang/aura/shared v0.0.0
	github.com/oneliang/aura/tools v0.0.0
	golang.org/x/sync v0.20.0
)

require (
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
)

replace (
	github.com/oneliang/aura/shared => ../shared
	github.com/oneliang/aura/tools => ../tools
)
