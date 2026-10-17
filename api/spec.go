package api

import _ "embed"

// the full contract, including operations served by plain handlers that code generation skips
//
//go:embed openapi.yaml
var OpenAPI []byte
