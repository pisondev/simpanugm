//go:build integration

package server_test

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type notificationList struct {
	Items []struct {
		ID     uuid.UUID `json:"id"`
		Event  string    `json:"event"`
		Title  string    `json:"title"`
		Body   string    `json:"body"`
		Link   string    `json:"link"`
		ReadAt *string   `json:"read_at"`
	} `json:"items"`
}

func unread(t *testing.T, c *testenv.Client) int {
	t.Helper()
	var n struct {
		Count int `json:"count"`
	}
	c.JSON(http.MethodGet, "/api/v1/notifications/unread-count", nil).Decode(t, &n)
	return n.Count
}

// files a found report with the finder and has an officer of the library post receive it
func receiveFound(t *testing.T, e *testenv.Env, finder testenv.User, officer testenv.User, key string) foundReport {
	t.Helper()
	library := e.Post("pos-perpustakaan")
	body := foundBody(e, e.Clock.Now(), "payung", []uuid.UUID{finder.Client.UploadID("found_report")}, map[string]any{"mode": "handover", "intended_post_id": library.String()})
	res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", body, "Idempotency-Key", key)
	if res.Status != http.StatusCreated {
		t.Fatalf("create found: %d %s", res.Status, res.Body)
	}
	var f foundReport
	res.Decode(t, &f)
	res = officer.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+f.Code+"/receive", map[string]any{"post_id": library.String(), "storage_slot": "Rak A1"})
	if res.Status != http.StatusOK {
		t.Fatalf("receive: %d %s", res.Status, res.Body)
	}
	return f
}

// US-30 criterion 2 and FR-NOTIF-01, 03 and 06
func TestFinderIsNotifiedWhenTheItemReachesAPost(t *testing.T) {
	e := testenv.New(t)
	e.Clock.Set(time.Date(2026, 11, 2, 3, 0, 0, 0, time.UTC)) // 10.00 WIB, outside the quiet hours
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	f := receiveFound(t, e, finder, officer, "notify-found-1")
	var list notificationList
	finder.Client.JSON(http.MethodGet, "/api/v1/notifications", nil).Decode(t, &list)
	if len(list.Items) != 1 || list.Items[0].Event != string(notify.FoundReceived) || list.Items[0].Link != "/laporan/temuan/"+f.Code ||
		!strings.Contains(list.Items[0].Body, "Pos Lost and Found Perpustakaan Pusat") {
		t.Fatalf("finder notifications: %+v", list.Items)
	}
	if unread(t, finder.Client) != 1 {
		t.Fatal("one unread notification expected")
	}
	if e.EmailCount(finder.Email, string(notify.FoundReceived)) != 1 {
		t.Fatal("found.received is an email event")
	}
	if at := e.EmailScheduledAt(finder.Email, string(notify.FoundReceived)); at.After(e.Clock.Now().Add(time.Minute)) {
		t.Fatalf("an email outside the quiet hours goes now, scheduled %s", at)
	}

	var other notificationList
	officer.Client.JSON(http.MethodGet, "/api/v1/notifications", nil).Decode(t, &other)
	if len(other.Items) != 0 {
		t.Fatal("a notification is visible only to its recipient")
	}
	if res := officer.Client.JSON(http.MethodPost, "/api/v1/notifications/"+list.Items[0].ID.String()+"/read", nil); res.Status != http.StatusNotFound {
		t.Fatalf("someone else's notification: %d", res.Status)
	}
	if res := finder.Client.JSON(http.MethodPost, "/api/v1/notifications/"+list.Items[0].ID.String()+"/read", nil); res.Status != http.StatusNoContent {
		t.Fatalf("mark read: %d %s", res.Status, res.Body)
	}
	if unread(t, finder.Client) != 0 {
		t.Fatal("read notifications are not counted")
	}
}

// BR-NTF-03 and FR-NOTIF-04
func TestQuietHoursAndPreferencesShapeTheEmail(t *testing.T) {
	e := testenv.New(t)
	e.Clock.Set(time.Date(2026, 11, 2, 16, 30, 0, 0, time.UTC)) // 23.30 WIB
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	receiveFound(t, e, finder, officer, "notify-quiet-1")
	want := time.Date(2026, 11, 3, 6, 0, 0, 0, clock.WIB())
	if at := e.EmailScheduledAt(finder.Email, string(notify.FoundReceived)); !at.Equal(want) {
		t.Fatalf("email during quiet hours scheduled %s, want 06.00 WIB", at.In(clock.WIB()))
	}

	res := finder.Client.JSON(http.MethodPut, "/api/v1/me/notification-preferences", map[string]any{
		"items": []map[string]any{{"group": "found", "app": true, "email": false}},
	})
	var prefs struct {
		Items []struct {
			Group string `json:"group"`
			App   bool   `json:"app"`
			Email bool   `json:"email"`
		} `json:"items"`
	}
	res.Decode(t, &prefs)
	if res.Status != http.StatusOK || len(prefs.Items) != 3 {
		t.Fatalf("preferences: %d %s", res.Status, res.Body)
	}
	for _, p := range prefs.Items {
		if p.Group == "found" && (p.Email || !p.App) || p.Group != "found" && (!p.Email || !p.App) {
			t.Fatalf("preference %+v", p)
		}
	}
	receiveFound(t, e, finder, officer, "notify-quiet-2")
	if n := e.EmailCount(finder.Email, string(notify.FoundReceived)); n != 1 {
		t.Fatalf("email switched off, still %d emails", n)
	}
	if unread(t, finder.Client) != 2 {
		t.Fatal("the app channel stays on")
	}
	bad := finder.Client.JSON(http.MethodPut, "/api/v1/me/notification-preferences", map[string]any{
		"items": []map[string]any{{"group": "account", "app": false, "email": false}},
	})
	if bad.Status != http.StatusBadRequest && bad.Status != http.StatusUnprocessableEntity {
		t.Fatalf("account events are mandatory and cannot be switched off: %d", bad.Status)
	}
}

// FR-NOTIF-02 and US-35 criteria 1 and 2
func TestStreamPushesNewNotificationsAndResumes(t *testing.T) {
	e := testenv.New(t)
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	res := finder.Client.Open("/api/v1/notifications/stream")
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	events := make(chan string, 8)
	go func() {
		sc := bufio.NewScanner(res.Body)
		var id string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "id: ") {
				id = strings.TrimPrefix(line, "id: ")
			}
			if line == "event: notification" {
				events <- id
			}
		}
	}()
	receiveFound(t, e, finder, officer, "notify-stream-1")
	var first string
	select {
	case first = <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("no notification event within 2 s")
	}

	_ = res.Body.Close()
	resume := finder.Client.Open("/api/v1/notifications/stream", "Last-Event-ID", first)
	defer func() { _ = resume.Body.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.InTx(func(tx pgx.Tx) error {
		return e.Services.Notifier.Send(ctx, tx, notify.Message{Event: notify.MatchHintForFinder, To: finder.ID, Link: "/laporan/temuan/x", Data: map[string]string{"found_code": "FND-0000-0000"}})
	}); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(resume.Body)
	deadline := time.After(3 * time.Second)
	got := make(chan bool, 1)
	go func() {
		for sc.Scan() {
			if sc.Text() == "event: notification" {
				got <- true
				return
			}
		}
	}()
	select {
	case <-got:
	case <-deadline:
		t.Fatal("a notification sent while the stream was closed arrives after resuming with Last-Event-ID")
	}
}

// BR-ACC-04: the badge and the stream are passive, an idle officer is still signed out
func TestPassiveRequestsDoNotKeepAnIdleSessionAlive(t *testing.T) {
	e := testenv.New(t)
	officer := e.User(access.Officer, e.Post("pos-pusat"))
	for range 3 {
		e.Clock.Advance(20 * time.Minute)
		if res := officer.Client.JSON(http.MethodGet, "/api/v1/notifications/unread-count", nil); res.Status != http.StatusOK && res.Status != http.StatusUnauthorized {
			t.Fatalf("unread count: %d", res.Status)
		}
	}
	e.Clock.Advance(2 * time.Minute)
	if res := officer.Client.JSON(http.MethodGet, "/api/v1/me", nil); res.Status != http.StatusUnauthorized {
		t.Fatalf("officer idle for 62 minutes with only badge polling: %d", res.Status)
	}
}

// BR-NTF-02 and US-36 criterion 3: one digest per officer, and none when there is nothing to tell
func TestOfficerDigestOnlyWhenItHasContent(t *testing.T) {
	e := testenv.New(t)
	busy := e.User(access.Officer, e.Post("pos-perpustakaan"))
	quiet := e.User(access.Officer, e.Post("pos-agro"))
	ctx := context.Background()
	for i, code := range []string{"CLM-2611-0001", "CLM-2611-0002"} {
		if err := e.InTx(func(tx pgx.Tx) error {
			return e.Services.Notifier.Send(ctx, tx, notify.Message{
				Event: notify.ClaimSubmitted, To: busy.ID, Link: "/petugas/klaim/" + code,
				Data: map[string]string{"claim_code": code, "found_code": "FND-2611-001" + string(rune('0'+i))},
			})
		}); err != nil {
			t.Fatal(err)
		}
	}
	if e.EmailCount(busy.Email, string(notify.ClaimSubmitted)) != 0 {
		t.Fatal("digest events never email directly")
	}
	e.Clock.Advance(time.Hour)
	w := &notify.DigestWorker{Q: sqlc.New(e.Pool), Jobs: e.Jobs, Clock: e.Clock, BaseURL: e.Config.PublicBaseURL}
	if err := w.Work(ctx, &river.Job[notify.DigestArgs]{}); err != nil {
		t.Fatal(err)
	}
	if e.EmailCount(busy.Email, "officer.digest") != 1 {
		t.Fatal("the busy officer gets one digest")
	}
	if a := e.LastEmail(busy.Email); !strings.Contains(a.Text, "CLM-2611-0001") || !strings.Contains(a.Text, "CLM-2611-0002") {
		t.Fatalf("digest text: %s", a.Text)
	}
	if e.EmailCount(quiet.Email, "officer.digest") != 0 {
		t.Fatal("an officer with nothing to do gets no digest")
	}
}
