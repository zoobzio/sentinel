module github.com/zoobzio/sentinel

go 1.23.1

require github.com/zoobzio/zlog v0.0.0

require (
	github.com/zoobzio/pipz v0.6.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/zoobzio/zlog => ../zlog

replace github.com/zoobzio/pipz => ../pipz
