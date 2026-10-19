//go:build integration

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

// US-42: Moderation of reports and users
func TestAdminReportAndUserModeration(t *testing.T) {
	e := isolated(t)
	admin := e.User(access.Admin)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))
	member := e.User(access.Member)

	l := fileLost(t, e, member, "mod-lost-01", nil)
	f := fileFound(t, e, member, "mod-found-01", nil)

	// Non-admin cannot hide reports
	res := member.Client.JSON(http.MethodPost, "/api/v1/admin/reports/lost/"+l.ID+"/hide", map[string]any{"reason": "konten tidak pantas"})
	if res.Status != http.StatusForbidden {
		t.Fatalf("member cannot hide report: %d", res.Status)
	}

	res = officer.Client.JSON(http.MethodPost, "/api/v1/admin/reports/lost/"+l.ID+"/hide", map[string]any{"reason": "konten tidak pantas"})
	if res.Status != http.StatusForbidden {
		t.Fatalf("officer cannot hide report: %d", res.Status)
	}

	// Admin hides lost report
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/reports/lost/"+l.ID+"/hide", map[string]any{"reason": "duplikat spam"})
	if res.Status != http.StatusOK {
		t.Fatalf("admin hides lost report: %d %s", res.Status, res.Body)
	}

	// Verify report in DB is hidden (hidden_at and hidden_reason set)
	var hiddenAt *time.Time
	var hiddenReason *string
	if err := e.Pool.QueryRow(context.Background(), "SELECT hidden_at, hidden_reason FROM lost_reports WHERE id = $1", l.ID).Scan(&hiddenAt, &hiddenReason); err != nil {
		t.Fatalf("query hidden: %v", err)
	}
	if hiddenAt == nil || hiddenReason == nil || *hiddenReason != "duplikat spam" {
		t.Fatalf("expected hidden_at set, got %v, reason %v", hiddenAt, hiddenReason)
	}

	// Admin unhides lost report
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/reports/lost/"+l.ID+"/unhide", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("admin unhides lost report: %d %s", res.Status, res.Body)
	}
	if err := e.Pool.QueryRow(context.Background(), "SELECT hidden_at FROM lost_reports WHERE id = $1", l.ID).Scan(&hiddenAt); err != nil {
		t.Fatalf("query hidden_at: %v", err)
	}
	if hiddenAt != nil {
		t.Fatalf("expected hidden_at nil after unhide, got %v", hiddenAt)
	}

	// Admin hides found report
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/reports/found/"+f.ID+"/hide", map[string]any{"reason": "barang terlarang"})
	if res.Status != http.StatusOK {
		t.Fatalf("admin hides found report: %d %s", res.Status, res.Body)
	}
	if err := e.Pool.QueryRow(context.Background(), "SELECT hidden_at, hidden_reason FROM found_reports WHERE id = $1", f.ID).Scan(&hiddenAt, &hiddenReason); err != nil {
		t.Fatalf("query hidden: %v", err)
	}
	if hiddenAt == nil || hiddenReason == nil || *hiddenReason != "barang terlarang" {
		t.Fatalf("expected hidden_at set, got %v, reason %v", hiddenAt, hiddenReason)
	}

	// Admin unhides found report
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/reports/found/"+f.ID+"/unhide", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("admin unhides found report: %d %s", res.Status, res.Body)
	}
	if err := e.Pool.QueryRow(context.Background(), "SELECT hidden_at FROM found_reports WHERE id = $1", f.ID).Scan(&hiddenAt); err != nil {
		t.Fatalf("query hidden_at: %v", err)
	}
	if hiddenAt != nil {
		t.Fatalf("expected hidden_at nil after unhide, got %v", hiddenAt)
	}

	// User moderation: suspend, revoke sessions, unsuspend, clear claim restriction
	targetUser := e.User(access.Member)
	// Suspend
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/users/"+targetUser.ID.String()+"/suspend", map[string]any{"reason": "pelanggaran ketentuan"})
	if res.Status != http.StatusOK {
		t.Fatalf("admin suspends user: %d %s", res.Status, res.Body)
	}

	// Suspended user cannot make authenticated requests
	res = targetUser.Client.JSON(http.MethodGet, "/api/v1/me", nil)
	if res.Status != http.StatusUnauthorized {
		t.Fatalf("suspended user session should be revoked: %d %s", res.Status, res.Body)
	}

	// Attempting to login while suspended
	loginRes := targetUser.Client.JSON(http.MethodPost, "/api/v1/auth/login", map[string]any{"email": targetUser.Email, "password": testenv.Password})
	if loginRes.Status != http.StatusUnauthorized {
		t.Fatalf("suspended user login must be unauthorized: %d %s", loginRes.Status, loginRes.Body)
	}

	// Unsuspend
	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/users/"+targetUser.ID.String()+"/unsuspend", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("admin unsuspends user: %d %s", res.Status, res.Body)
	}

	// User can log in again
	loginRes = targetUser.Client.JSON(http.MethodPost, "/api/v1/auth/login", map[string]any{"email": targetUser.Email, "password": testenv.Password})
	if loginRes.Status != http.StatusOK {
		t.Fatalf("unsuspended user should be able to log in: %d %s", loginRes.Status, loginRes.Body)
	}

	// Clear claim restriction
	past := time.Now().Add(-1 * time.Hour)
	_, err := e.Pool.Exec(context.Background(), "UPDATE users SET claim_restricted_at = $1 WHERE id = $2", past, targetUser.ID)
	if err != nil {
		t.Fatalf("set claim_restricted_at: %v", err)
	}

	res = admin.Client.JSON(http.MethodPost, "/api/v1/admin/users/"+targetUser.ID.String()+"/clear-claim-restriction", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("clear claim restriction: %d %s", res.Status, res.Body)
	}

	var restrictedAt *time.Time
	if err := e.Pool.QueryRow(context.Background(), "SELECT claim_restricted_at FROM users WHERE id = $1", targetUser.ID).Scan(&restrictedAt); err != nil {
		t.Fatalf("query claim_restricted_at: %v", err)
	}
	if restrictedAt != nil {
		t.Fatalf("claim restriction was not cleared")
	}
}

// US-43: Runtime settings
func TestAdminRuntimeSettings(t *testing.T) {
	e := isolated(t)
	admin := e.User(access.Admin)
	member := e.User(access.Member)

	t.Cleanup(func() {
		_ = admin.Client.JSON(http.MethodPatch, "/api/v1/admin/settings", map[string]any{
			"settings": []map[string]any{
				{"key": "claim.max_open", "value": 3},
			},
		})
	})

	// Non-admin forbidden
	res := member.Client.JSON(http.MethodGet, "/api/v1/admin/settings", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("member GET settings: %d", res.Status)
	}

	// Admin GET settings
	res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/settings", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("admin GET settings: %d %s", res.Status, res.Body)
	}
	var list struct {
		Items []struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		} `json:"items"`
	}
	res.Decode(t, &list)
	if len(list.Items) == 0 {
		t.Fatalf("expected settings list, got empty")
	}

	// PATCH setting out of bounds
	res = admin.Client.JSON(http.MethodPatch, "/api/v1/admin/settings", map[string]any{
		"settings": []map[string]any{
			{"key": "claim.max_open", "value": -5},
		},
	})
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for out-of-range setting: %d %s", res.Status, res.Body)
	}

	// PATCH setting within valid bounds
	res = admin.Client.JSON(http.MethodPatch, "/api/v1/admin/settings", map[string]any{
		"settings": []map[string]any{
			{"key": "claim.max_open", "value": 5},
		},
	})
	if res.Status != http.StatusOK {
		t.Fatalf("patch valid setting: %d %s", res.Status, res.Body)
	}

	// Verify setting persisted
	res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/settings", nil)
	res.Decode(t, &list)
	found := false
	for _, it := range list.Items {
		if it.Key == "claim.max_open" {
			found = true
			if string(it.Value) != "5" {
				t.Fatalf("expected claim.max_open to be 5, got %s", string(it.Value))
			}
		}
	}
	if !found {
		t.Fatalf("claim.max_open not found in settings list")
	}
}

// US-44: Audit log filtering, CSV export, and immutability
func TestAdminAuditLogsAndImmutability(t *testing.T) {
	e := isolated(t)
	admin := e.User(access.Admin)
	member := e.User(access.Member)

	// Non-admin forbidden
	res := member.Client.JSON(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	if res.Status != http.StatusForbidden {
		t.Fatalf("member GET audit-logs: %d", res.Status)
	}

	// Admin GET audit-logs JSON
	res = admin.Client.JSON(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("admin GET audit-logs: %d %s", res.Status, res.Body)
	}

	// Admin GET audit-logs CSV streaming via ?format=csv
	csvRes := admin.Client.Do(http.MethodGet, "/api/v1/admin/audit-logs?format=csv", nil, "")
	if csvRes.Status != http.StatusOK {
		t.Fatalf("audit log CSV status: %d %s", csvRes.Status, csvRes.Body)
	}
	if !strings.HasPrefix(string(csvRes.Body), "id,occurred_at,actor_id,actor_name,actor_role,action,entity_type,entity_id,changes,request_id,ip,user_agent") {
		t.Fatalf("unexpected CSV header: %s", string(csvRes.Body))
	}

	// Immutability check (US-44 criterion 2):
	// simpan_app user must NOT have permission to UPDATE or DELETE or TRUNCATE audit_logs
	_, updateErr := e.Pool.Exec(context.Background(), "UPDATE audit_logs SET action = 'tampered'")
	if updateErr == nil {
		t.Fatalf("expected UPDATE on audit_logs to fail with permission denied, got nil")
	}
	if !strings.Contains(updateErr.Error(), "42501") && !strings.Contains(updateErr.Error(), "permission denied") {
		t.Fatalf("expected permission denied error on UPDATE, got: %v", updateErr)
	}

	_, deleteErr := e.Pool.Exec(context.Background(), "DELETE FROM audit_logs")
	if deleteErr == nil {
		t.Fatalf("expected DELETE on audit_logs to fail with permission denied, got nil")
	}
	if !strings.Contains(deleteErr.Error(), "42501") && !strings.Contains(deleteErr.Error(), "permission denied") {
		t.Fatalf("expected permission denied error on DELETE, got: %v", deleteErr)
	}
}

// US-45: Account self-deletion with BR-RET-05
func TestAccountSelfDeletion(t *testing.T) {
	e := isolated(t)
	s := newScene(t, e)

	// User with approved claim cannot delete account (BR-RET-05)
	s.lost = fileLost(t, e, s.owner, "del-lost-appr", nil)
	s.found = fileFound(t, e, s.finder, "del-found-appr", nil)
	s.receive(t, s.found, s.officer)
	claim := s.claimOn(t, s.found, s.lost, "del-claim-appr")

	// Set claim status to approved in database
	_, err := e.Pool.Exec(context.Background(), "UPDATE claims SET status = 'approved' WHERE id = $1", claim.ID)
	if err != nil {
		t.Fatalf("update claim status: %v", err)
	}

	// Owner attempts deletion: must be rejected
	res := s.owner.Client.JSON(http.MethodPost, "/api/v1/me/deletion", nil)
	if res.Status != http.StatusConflict && res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("user with approved claim should not be able to delete account: %d %s", res.Status, res.Body)
	}

	// New user without approved claims can delete account
	userToDelete := e.User(access.Member)
	userLost := fileLost(t, e, userToDelete, "del-lost-ok", nil)
	userFound := fileFound(t, e, s.finder, "del-found-ok", nil)
	s.receive(t, userFound, s.officer)

	// Open a claim
	userClaim := userToDelete.Client.JSON(http.MethodPost, "/api/v1/claims", claimBody(userFound.ID, userLost.ID, s.walletAnswers()), "Idempotency-Key", "del-claim-open")
	if userClaim.Status != http.StatusCreated {
		t.Fatalf("open claim: %d %s", userClaim.Status, userClaim.Body)
	}

	// Now delete account
	delRes := userToDelete.Client.JSON(http.MethodPost, "/api/v1/me/deletion", nil)
	if delRes.Status != http.StatusOK && delRes.Status != http.StatusNoContent {
		t.Fatalf("deletion failed: %d %s", delRes.Status, delRes.Body)
	}

	// Old session is revoked
	meRes := userToDelete.Client.JSON(http.MethodGet, "/api/v1/me", nil)
	if meRes.Status != http.StatusUnauthorized {
		t.Fatalf("deleted user session should be revoked: %d %s", meRes.Status, meRes.Body)
	}

	// Verify user anonymized in DB
	var fullName, email string
	var phone, nimNip *string
	if err := e.Pool.QueryRow(context.Background(), "SELECT full_name, email, phone, nim_nip FROM users WHERE id = $1", userToDelete.ID).Scan(&fullName, &email, &phone, &nimNip); err != nil {
		t.Fatalf("query user: %v", err)
	}
	if fullName != "Pengguna terhapus" {
		t.Fatalf("expected full_name Pengguna terhapus, got %s", fullName)
	}
	if phone != nil || nimNip != nil {
		t.Fatalf("phone and nim_nip must be cleared")
	}

	// Verify open lost report closed with account_deleted
	var lostStatus string
	var closeReason *string
	if err := e.Pool.QueryRow(context.Background(), "SELECT status, close_reason FROM lost_reports WHERE id = $1", userLost.ID).Scan(&lostStatus, &closeReason); err != nil {
		t.Fatalf("query lost report: %v", err)
	}
	if lostStatus != "closed" || closeReason == nil || *closeReason != "account_deleted" {
		t.Fatalf("lost report status=%s closeReason=%v", lostStatus, closeReason)
	}

	// Verify open claim withdrawn
	var claimStatus string
	if err := e.Pool.QueryRow(context.Background(), "SELECT status FROM claims WHERE claimant_id = $1", userToDelete.ID).Scan(&claimStatus); err != nil {
		t.Fatalf("query claim: %v", err)
	}
	if claimStatus != "withdrawn" {
		t.Fatalf("expected claim status withdrawn, got %s", claimStatus)
	}
}
