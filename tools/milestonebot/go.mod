// milestonebot is a nested module on purpose. It is repository tooling, not
// part of the toolkit, and keeping it in its own module means `go get
// github.com/go-mizu/mizu` never pulls a YAML parser along with it.
module github.com/go-mizu/mizu/tools/milestonebot

go 1.26

require gopkg.in/yaml.v3 v3.0.1
