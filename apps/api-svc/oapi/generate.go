//go:generate go tool oapi-codegen -o server.gen.go --config=server.cfg.yaml openapi.yaml
//go:generate go tool oapi-codegen -o types.gen.go --config=types.cfg.yaml openapi.yaml

// Package oapi contains code generated from oapi/openapi.yaml.
package oapi
