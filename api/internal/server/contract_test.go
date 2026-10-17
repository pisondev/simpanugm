package server

import (
	"context"
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	apispec "github.com/pisondev/simpanugm/api"
	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

func TestEveryOperationHasExactlyOneRule(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(apispec.OpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	inSpec := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.OperationID == "" {
				t.Errorf("%s %s has no operationId", method, path)
				continue
			}
			inSpec[op.OperationID] = true
			rule, ok := access.Operations[op.OperationID]
			if !ok {
				t.Errorf("operation %s has no access rule", op.OperationID)
				continue
			}
			public := op.Security != nil && len(*op.Security) == 0
			if public != rule.Public {
				t.Errorf("operation %s: spec public=%v, rule public=%v", op.OperationID, public, rule.Public)
			}
		}
	}
	for id := range access.Operations {
		if !inSpec[id] {
			t.Errorf("rule %s has no operation in openapi.yaml", id)
		}
	}
}

func TestCheck(t *testing.T) {
	member := &access.Principal{Role: access.Member}
	verified := &access.Principal{Role: access.Member, EmailVerified: true}
	officer := &access.Principal{Role: access.Officer, EmailVerified: true}
	cases := []struct {
		name string
		p    *access.Principal
		op   string
		code string
	}{
		{"public needs nobody", nil, "login", ""},
		{"private needs a session", nil, "getMe", "unauthenticated"},
		{"unverified cannot report", member, "createLostReport", "account_unverified"},
		{"verified member reports", verified, "createLostReport", ""},
		{"members cannot search", verified, "searchReports", "forbidden"},
		{"officers search", officer, "searchReports", ""},
		{"officers do not manage users", officer, "adminInviteUser", "forbidden"},
		{"unknown operations are denied", officer, "dropDatabase", "forbidden"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := access.Check(c.p, c.op)
			got := ""
			var ae *apperr.Error
			if errors.As(err, &ae) {
				got = ae.Code
			}
			if got != c.code {
				t.Fatalf("Check = %q, want %q", got, c.code)
			}
		})
	}
}

func TestOperationIDNormalisation(t *testing.T) {
	if OperationID("CreateLostReport") != "createLostReport" || OperationID("") != "" {
		t.Fatal("normalisation")
	}
}
