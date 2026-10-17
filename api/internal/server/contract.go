package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/httpx"
)

// permission rule first, then schema validation, ADR-0009
func Contract(doc *openapi3.T) (func(http.Handler) http.Handler, error) {
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, err
	}
	opts := &openapi3filter.Options{
		AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		MultiError:         true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, params, err := router.FindRoute(r)
			if err != nil {
				if errors.Is(err, routers.ErrMethodNotAllowed) {
					httpx.WriteProblem(w, r, apperr.New(http.StatusMethodNotAllowed, "method_not_allowed", "Metode tidak didukung", "Metode ini tidak tersedia untuk alamat tersebut."))
					return
				}
				httpx.WriteProblem(w, r, apperr.NotFound())
				return
			}
			if err := access.Check(access.FromContext(r.Context()), OperationID(route.Operation.OperationID)); err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			in := &openapi3filter.RequestValidationInput{Request: r, PathParams: params, Route: route, Options: opts}
			if err := openapi3filter.ValidateRequest(r.Context(), in); err != nil {
				httpx.WriteProblem(w, r, schemaProblem(err))
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

// oapi-codegen rewrites ids to PascalCase in what it generates, openapi.yaml and the rule table use camelCase
func OperationID(id string) string {
	if id == "" {
		return id
	}
	return strings.ToLower(id[:1]) + id[1:]
}

func schemaProblem(err error) *apperr.Error {
	var fields []apperr.FieldError
	var walk func(error)
	walk = func(e error) {
		var multi openapi3.MultiError
		if errors.As(e, &multi) {
			for _, m := range multi {
				walk(m)
			}
			return
		}
		var re *openapi3filter.RequestError
		if errors.As(e, &re) {
			field := ""
			if re.Parameter != nil {
				field = re.Parameter.Name
			}
			var se *openapi3.SchemaError
			if errors.As(re.Err, &se) {
				if p := se.JSONPointer(); len(p) > 0 {
					field = strings.Join(p, ".")
				}
				fields = append(fields, apperr.FieldError{Field: field, Code: "schema_" + se.SchemaField, Message: "Isian tidak sesuai format"})
				return
			}
			var multi openapi3.MultiError
			if errors.As(re.Err, &multi) {
				walk(multi)
				return
			}
			fields = append(fields, apperr.FieldError{Field: field, Code: "invalid", Message: "Isian tidak sesuai format"})
			return
		}
		var se *openapi3.SchemaError
		if errors.As(e, &se) {
			fields = append(fields, apperr.FieldError{Field: strings.Join(se.JSONPointer(), "."), Code: "schema_" + se.SchemaField, Message: "Isian tidak sesuai format"})
			return
		}
		fields = append(fields, apperr.FieldError{Field: "", Code: "invalid", Message: "Isian tidak sesuai format"})
	}
	walk(err)
	if len(fields) == 0 {
		return apperr.BadRequest("Permintaan tidak sesuai kontrak API.")
	}
	return apperr.Validation(fields...)
}
