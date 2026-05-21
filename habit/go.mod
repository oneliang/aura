module github.com/oneliang/aura/habit

go 1.26.1

require (
	github.com/google/uuid v1.6.0
	github.com/oneliang/aura/shared v0.0.0
)

replace (
	github.com/oneliang/aura/shared => ../shared
	github.com/oneliang/aura/storage => ../storage
)
