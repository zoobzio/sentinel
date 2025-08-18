module github.com/zoobzio/sentinel

go 1.23.1

require (
	github.com/zoobzio/pipz v1.0.0
	github.com/zoobzio/zlog v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/time v0.12.0 // indirect

replace github.com/zoobzio/zlog => ../zlog
