module github.com/go-msvc/config

go 1.12

require (
	github.com/go-msvc/errors v0.0.0-20191116111408-1c2c4914594f
	github.com/go-msvc/log v0.0.0-20200515104948-e039d1c2f30d
	github.com/go-msvc/logger v0.0.0-20200921071849-c0ba6025fb9f
	github.com/pkg/errors v0.9.1
)

replace github.com/go-msvc/errors => ../errors
replace github.com/go-msvc/logger => ../logger
