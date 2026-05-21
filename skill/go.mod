module github.com/oneliang/aura/skill

go 1.26.1

require github.com/oneliang/aura/shared v0.0.0

require (
	github.com/kr/pretty v0.3.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/oneliang/aura/shared => ../shared
	github.com/oneliang/aura/skill => ./
)
