//go:build integration

package server_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/testenv"
)

type candidateLine struct {
	Level   *string  `json:"level"`
	Reason  string   `json:"reason"`
	Chips   []string `json:"chips"`
	Missing bool     `json:"missing"`
}

type candidate struct {
	ID      uuid.UUID `json:"id"`
	Rank    *int      `json:"rank"`
	Tier    string    `json:"tier"`
	Overall float64   `json:"overall"`
	Status  string    `json:"status"`
	Found   struct {
		Code          string  `json:"code"`
		HiddenDetails *string `json:"hidden_details"`
		Photos        []struct {
			MediaID uuid.UUID `json:"media_id"`
			Variant string    `json:"variant"`
		} `json:"photos"`
	} `json:"found_report"`
	Lost struct {
		Code      string  `json:"code"`
		OwnerName *string `json:"owner_name"`
	} `json:"lost_report"`
	Explanation struct {
		VisualMode string        `json:"visual_mode"`
		Visual     candidateLine `json:"visual"`
		Text       candidateLine `json:"text"`
		Location   candidateLine `json:"location"`
		Time       candidateLine `json:"time"`
		Category   candidateLine `json:"category"`
	} `json:"explanation"`
	Scores *struct {
		Visual   *float64 `json:"visual"`
		Semantic *float64 `json:"semantic"`
	} `json:"scores"`
	Flags []string `json:"flags"`
}

type candidateList struct {
	Items []candidate `json:"items"`
}

var era atomic.Int64

// each matching test lives in its own stretch of time, so reports other tests left in the shared database never become
// its candidates. The clock moves before any user exists, so sessions follow it
func isolated(t *testing.T, opts ...testenv.Option) *testenv.Env {
	e := testenv.New(t, opts...)
	e.Clock.Set(time.Date(2020, 1, 6, 2, 0, 0, 0, time.UTC).AddDate(0, 0, int(era.Add(1))*150))
	return e
}

func candidates(t *testing.T, c *testenv.Client, path string) candidateList {
	t.Helper()
	res := c.JSON(http.MethodGet, path, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, res.Status, res.Body)
	}
	var l candidateList
	res.Decode(t, &l)
	return l
}

func fileLost(t *testing.T, e *testenv.Env, owner testenv.User, key string, edit func(map[string]any)) lostReport {
	t.Helper()
	body := lostBody(e, e.Clock.Now(), nil)
	if edit != nil {
		edit(body)
	}
	res := owner.Client.JSON(http.MethodPost, "/api/v1/lost-reports", body, "Idempotency-Key", key)
	if res.Status != http.StatusCreated {
		t.Fatalf("create lost: %d %s", res.Status, res.Body)
	}
	var r lostReport
	res.Decode(t, &r)
	return r
}

// a brown wallet handed over to the library post, the counterpart of lostBody
func fileFound(t *testing.T, e *testenv.Env, finder testenv.User, key string, edit func(map[string]any)) foundReport {
	t.Helper()
	library := e.Post("pos-perpustakaan")
	body := foundBody(e, e.Clock.Now(), "dompet", []uuid.UUID{finder.Client.UploadID("found_report")}, map[string]any{"mode": "handover", "intended_post_id": library.String()})
	body["item_name"], body["colours"] = "Dompet cokelat", []string{"cokelat"}
	if edit != nil {
		edit(body)
	}
	res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports", body, "Idempotency-Key", key)
	if res.Status != http.StatusCreated {
		t.Fatalf("create found: %d %s", res.Status, res.Body)
	}
	var f foundReport
	res.Decode(t, &f)
	return f
}

func score(t *testing.T, e *testenv.Env) {
	t.Helper()
	for _, err := range testenv.Drain(e, &matching.MatchReportWorker{E: e.Services.Engine}) {
		if err != nil {
			t.Fatalf("match.report: %v", err)
		}
	}
}

func lostStatus(t *testing.T, c *testenv.Client, code string) (string, int) {
	t.Helper()
	var r struct {
		Status         string `json:"status"`
		CandidateCount int    `json:"candidate_count"`
	}
	c.JSON(http.MethodGet, "/api/v1/lost-reports/"+code, nil).Decode(t, &r)
	return r.Status, r.CandidateCount
}

func foundStatus(t *testing.T, c *testenv.Client, code string) string {
	t.Helper()
	var r foundReport
	c.JSON(http.MethodGet, "/api/v1/found-reports/"+code, nil).Decode(t, &r)
	return r.Status
}

func events(t *testing.T, c *testenv.Client) []string {
	t.Helper()
	var l notificationList
	c.JSON(http.MethodGet, "/api/v1/notifications", nil).Decode(t, &l)
	var out []string
	for _, n := range l.Items {
		out = append(out, n.Event)
	}
	return out
}

func count(list []string, v string) int {
	n := 0
	for _, s := range list {
		if s == v {
			n++
		}
	}
	return n
}

// US-19, US-20, US-14, BR-VIS 5.1 and 5.2 with the structured signals only, the way matching runs without inference
func TestCandidatesAreRankedExplainedAndProjectedByRole(t *testing.T) {
	e := isolated(t)
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	stranger := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	lost := fileLost(t, e, owner, "match-lost-0001", nil)
	found := fileFound(t, e, finder, "match-found-0001", nil)
	if n := e.JobCount("match.report"); n != 2 {
		t.Fatalf("both reports queue a scoring job, got %d", n)
	}
	score(t, e)

	l := candidates(t, owner.Client, "/api/v1/lost-reports/"+lost.Code+"/matches")
	if len(l.Items) != 1 {
		t.Fatalf("owner candidates: %+v", l)
	}
	c := l.Items[0]
	if c.Tier != "high" || c.Rank == nil || *c.Rank != 1 || c.Found.Code != found.Code || c.Overall < 0.72 {
		t.Fatalf("candidate: %+v", c)
	}
	x := c.Explanation
	if x.VisualMode != "none" || !x.Visual.Missing || x.Location.Reason != "Ditemukan di lokasi yang sama." ||
		x.Time.Reason != "Ditemukan 1 jam setelah perkiraan waktu hilang." || x.Category.Reason != "Kategori sama." ||
		strings.Join(x.Text.Chips, "|") != "Warna sama: cokelat" {
		t.Fatalf("explanation: %+v", x)
	}
	// BR-VIS-04: an owner never receives component numbers, hidden details, flags or the finder
	raw := string(owner.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+lost.Code+"/matches", nil).Body)
	for _, leak := range []string{`"scores"`, `"hidden_details"`, `"flags"`, `"component_versions"`, "Rani", `"owner_name"`, finder.FullName} {
		if strings.Contains(raw, leak) {
			t.Fatalf("owner projection leaks %q: %s", leak, raw)
		}
	}
	if status, n := lostStatus(t, owner.Client, lost.Code); status != "potential_match" || n != 1 {
		t.Fatalf("L2: %s with %d candidates", status, n)
	}
	if s := foundStatus(t, finder.Client, found.Code); s != "potential_match" {
		t.Fatalf("F2: %s", s)
	}

	// the owner sees the display variant of the candidate photo, never the original, and nobody else gains access
	photo := c.Found.Photos[0]
	if photo.Variant != "display" {
		t.Fatalf("variant: %+v", photo)
	}
	media := "/api/v1/media/" + photo.MediaID.String()
	if res := owner.Client.Do(http.MethodGet, media+"?variant=display", nil, ""); res.Status != http.StatusOK {
		t.Fatalf("owner display: %d", res.Status)
	}
	if res := owner.Client.Do(http.MethodGet, media+"?variant=original", nil, ""); res.Status != http.StatusNotFound {
		t.Fatalf("owner original: %d", res.Status)
	}
	if res := stranger.Client.Do(http.MethodGet, media+"?variant=display", nil, ""); res.Status != http.StatusNotFound {
		t.Fatalf("stranger display: %d", res.Status)
	}

	// officers get the numbers and the owner, members outside the pair get nothing
	ol := candidates(t, officer.Client, "/api/v1/found-reports/"+found.Code+"/matches")
	if len(ol.Items) != 1 || ol.Items[0].Scores == nil || ol.Items[0].Lost.OwnerName == nil || ol.Items[0].Flags == nil {
		t.Fatalf("officer projection: %+v", ol)
	}
	var detail candidate
	officer.Client.JSON(http.MethodGet, "/api/v1/matches/"+c.ID.String(), nil).Decode(t, &detail)
	if detail.Found.HiddenDetails == nil {
		t.Fatalf("officer detail carries hidden details: %+v", detail)
	}
	if res := stranger.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+lost.Code+"/matches", nil); res.Status != http.StatusNotFound {
		t.Fatalf("stranger list: %d", res.Status)
	}
	if res := stranger.Client.JSON(http.MethodGet, "/api/v1/matches/"+c.ID.String(), nil); res.Status != http.StatusNotFound {
		t.Fatalf("stranger detail: %d", res.Status)
	}
	if res := finder.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+found.Code+"/matches", nil); res.Status != http.StatusForbidden {
		t.Fatalf("finder list: %d", res.Status)
	}

	// US-14: the finder learns the item name and category, nothing about the owner
	res := finder.Client.JSON(http.MethodGet, "/api/v1/found-reports/"+found.Code+"/match-hint", nil)
	var hint struct {
		Items []map[string]string `json:"items"`
	}
	res.Decode(t, &hint)
	if len(hint.Items) != 1 || hint.Items[0]["item_name"] != "Dompet kulit cokelat" || hint.Items[0]["category_name"] != "Dompet" || len(hint.Items[0]) != 2 {
		t.Fatalf("hint: %s", res.Body)
	}
	for _, leak := range []string{owner.FullName, "Gantungan", "KTM dan kartu", lost.Code} {
		if strings.Contains(string(res.Body), leak) {
			t.Fatalf("hint leaks %q: %s", leak, res.Body)
		}
	}
}

// US-22 and BR-NTF-01: the owner hears about a strong candidate once per window, the finder and the post right away
func TestStrongCandidatesNotifyWithinTheThrottle(t *testing.T) {
	e := isolated(t)
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))
	notifyWorker := &matching.MatchNotifyWorker{E: e.Services.Engine}

	lost := fileLost(t, e, owner, "notify-lost-0001", nil)
	fileFound(t, e, finder, "notify-found-0001", nil)
	score(t, e)
	if n := e.JobCount("match.notify"); n != 1 {
		t.Fatalf("one owner notification job, got %d", n)
	}
	for _, err := range testenv.Drain(e, notifyWorker) {
		if err != nil {
			t.Fatalf("match.notify: %v", err)
		}
	}
	if ev := events(t, owner.Client); count(ev, "match.high_for_owner") != 1 {
		t.Fatalf("owner events: %v", ev)
	}
	if ev := events(t, finder.Client); count(ev, "match.hint_for_finder") != 1 {
		t.Fatalf("finder events: %v", ev)
	}
	if ev := events(t, officer.Client); count(ev, "match.high_for_officer") != 1 {
		t.Fatalf("officer events: %v", ev)
	}

	// a second strong candidate inside the window waits, then both are folded into one notification
	fileFound(t, e, finder, "notify-found-0002", func(b map[string]any) {
		b["found_location_id"] = e.Location("pusat-jajanan-lembah").String()
	})
	score(t, e)
	results := testenv.Drain(e, notifyWorker)
	var snooze *river.JobSnoozeError
	if len(results) != 1 || !errors.As(results[0], &snooze) || snooze.Duration <= 0 || snooze.Duration > 6*time.Hour {
		t.Fatalf("inside the window the job waits: %v", results)
	}
	if ev := events(t, owner.Client); count(ev, "match.high_for_owner") != 1 {
		t.Fatalf("still one owner notification: %v", ev)
	}
	e.Clock.Advance(6 * time.Hour)
	if _, err := e.Pool.Exec(context.Background(), "UPDATE river_job SET scheduled_at = now() WHERE kind = 'match.notify' AND state = 'scheduled'"); err != nil {
		t.Fatal(err)
	}
	for _, err := range testenv.Drain(e, notifyWorker) {
		if err != nil {
			t.Fatalf("match.notify after the window: %v", err)
		}
	}
	if ev := events(t, owner.Client); count(ev, "match.high_for_owner") != 2 {
		t.Fatalf("the folded notification arrives: %v", ev)
	}
	// rescoring without anything new queues no further notification
	if err := e.Services.Engine.ScoreReport(context.Background(), matching.TypeLost, uuid.MustParse(lost.ID)); err != nil {
		t.Fatal(err)
	}
	if n := e.JobCount("match.notify"); n != 0 {
		t.Fatalf("nothing new to tell: %d", n)
	}
}

// US-21, M2, M4 and US-19 criterion 4: dismissals, closing and rescoring settle both reports and never revive settled rows
func TestDismissalsAndClosingSettleStatuses(t *testing.T) {
	e := isolated(t)
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	lost := fileLost(t, e, owner, "dismiss-lost-0001", nil)
	found := fileFound(t, e, finder, "dismiss-found-0001", nil)
	score(t, e)
	c := candidates(t, owner.Client, "/api/v1/lost-reports/"+lost.Code+"/matches").Items[0]

	if res := owner.Client.JSON(http.MethodPost, "/api/v1/matches/"+c.ID.String()+"/feedback", map[string]string{"verdict": "relevant"}); res.Status != http.StatusNoContent {
		t.Fatalf("feedback: %d %s", res.Status, res.Body)
	}
	if res := owner.Client.JSON(http.MethodPost, "/api/v1/matches/"+c.ID.String()+"/dismiss", map[string]string{"reason": "Bukan dompet saya"}); res.Status != http.StatusNoContent {
		t.Fatalf("owner dismiss: %d %s", res.Status, res.Body)
	}
	if l := candidates(t, owner.Client, "/api/v1/lost-reports/"+lost.Code+"/matches"); len(l.Items) != 0 {
		t.Fatalf("dismissed for the owner: %+v", l)
	}
	if status, _ := lostStatus(t, owner.Client, lost.Code); status != "reported" {
		t.Fatalf("L3 after the owner dismissal: %s", status)
	}
	if l := candidates(t, officer.Client, "/api/v1/found-reports/"+found.Code+"/matches"); len(l.Items) != 1 {
		t.Fatalf("an owner dismissal keeps the pair for officers: %+v", l)
	}
	if res := owner.Client.JSON(http.MethodPost, "/api/v1/matches/"+c.ID.String()+"/dismiss", map[string]string{}); res.Status != http.StatusNotFound {
		t.Fatalf("second owner dismissal: %d", res.Status)
	}

	if res := officer.Client.JSON(http.MethodPost, "/api/v1/matches/"+c.ID.String()+"/dismiss", map[string]string{"reason": "Merek berbeda"}); res.Status != http.StatusNoContent {
		t.Fatalf("officer dismiss: %d %s", res.Status, res.Body)
	}
	if l := candidates(t, officer.Client, "/api/v1/found-reports/"+found.Code+"/matches"); len(l.Items) != 0 {
		t.Fatalf("an officer dismissal hides it for everyone: %+v", l)
	}
	if s := foundStatus(t, finder.Client, found.Code); s != "reported" {
		t.Fatalf("F3 after the officer dismissal: %s", s)
	}
	var labels int
	if err := e.Pool.QueryRow(context.Background(), "SELECT count(*) FROM match_feedback WHERE match_id = $1", c.ID).Scan(&labels); err != nil || labels != 3 {
		t.Fatalf("labels: %d %v", labels, err)
	}

	// an officer rematch rescoring the pair leaves the dismissed row alone
	if res := officer.Client.JSON(http.MethodPost, "/api/v1/reports/lost/"+lost.Code+"/rematch", nil); res.Status != http.StatusAccepted {
		t.Fatalf("rematch: %d %s", res.Status, res.Body)
	}
	if res := owner.Client.JSON(http.MethodPost, "/api/v1/reports/lost/"+lost.Code+"/rematch", nil); res.Status != http.StatusForbidden {
		t.Fatalf("members cannot rematch: %d", res.Status)
	}
	score(t, e)
	var status string
	if err := e.Pool.QueryRow(context.Background(), "SELECT status FROM matches WHERE id = $1", c.ID).Scan(&status); err != nil || status != "dismissed" {
		t.Fatalf("dismissed row after rescoring: %s %v", status, err)
	}
	// the nightly sweep queues the same rescoring for every open report
	if _, err := e.Jobs.Insert(context.Background(), jobargs.MatchSweep{}, nil); err != nil {
		t.Fatal(err)
	}
	for _, err := range testenv.Drain(e, &matching.MatchSweepWorker{E: e.Services.Engine}) {
		if err != nil {
			t.Fatalf("sweep: %v", err)
		}
	}
	var queued int
	if err := e.Pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job WHERE kind = 'match.report' AND state = 'available' AND args->>'report_id' = ANY($1)",
		[]string{lost.ID, found.ID}).Scan(&queued); err != nil || queued != 2 {
		t.Fatalf("sweep queued %d %v", queued, err)
	}
	if err := e.Services.Engine.ScoreReport(context.Background(), matching.TypeFound, uuid.MustParse(found.ID)); err != nil {
		t.Fatal(err)
	}
	if err := e.Pool.QueryRow(context.Background(), "SELECT status FROM matches WHERE id = $1", c.ID).Scan(&status); err != nil || status != "dismissed" {
		t.Fatalf("dismissed row after the sweep: %s %v", status, err)
	}
	if _, err := e.Pool.Exec(context.Background(), "DELETE FROM river_job WHERE kind = 'match.report' AND state = 'available' AND created_at >= $1 AND NOT (args->>'report_id' = ANY($2))",
		e.Started, []string{lost.ID, found.ID}); err != nil {
		t.Fatal(err)
	}

	// closing a found report supersedes its pairs, the lost report returns to reported once none is left
	lost2 := fileLost(t, e, owner, "dismiss-lost-0002", nil)
	found2 := fileFound(t, e, finder, "dismiss-found-0002", nil)
	score(t, e)
	if status, n := lostStatus(t, owner.Client, lost2.Code); status != "potential_match" || n != 2 {
		t.Fatalf("both found reports are candidates of the new lost report: %s %d", status, n)
	}
	for i, code := range []string{found2.Code, found.Code} {
		res := finder.Client.JSON(http.MethodPost, "/api/v1/found-reports/"+code+"/close", map[string]string{"reason": "returned_directly", "note": "Sudah dikembalikan langsung ke pemilik."})
		if res.Status != http.StatusOK {
			t.Fatalf("close found: %d %s", res.Status, res.Body)
		}
		score(t, e)
		want, wantN := "potential_match", 1
		if i == 1 {
			want, wantN = "reported", 0
		}
		if status, n := lostStatus(t, owner.Client, lost2.Code); status != want || n != wantN {
			t.Fatalf("M2 after closing %s: %s %d", code, status, n)
		}
	}
}

// BR-VIS-01: a weak candidate is stored but only officers who ask see it
func TestLowCandidatesStayWithOfficers(t *testing.T) {
	e := isolated(t)
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))

	lost := fileLost(t, e, owner, "low-lost-0001", nil)
	fileFound(t, e, finder, "low-found-0001", func(b map[string]any) { b["colours"] = []string{"hitam"} })
	score(t, e)
	if l := candidates(t, owner.Client, "/api/v1/lost-reports/"+lost.Code+"/matches"); len(l.Items) != 0 {
		t.Fatalf("owner sees no low candidate: %+v", l)
	}
	if l := candidates(t, officer.Client, "/api/v1/lost-reports/"+lost.Code+"/matches"); len(l.Items) != 0 {
		t.Fatalf("officers do not see low by default: %+v", l)
	}
	l := candidates(t, officer.Client, "/api/v1/lost-reports/"+lost.Code+"/matches?include_low=true")
	if len(l.Items) != 1 || l.Items[0].Tier != "low" || l.Items[0].Explanation.Text.Reason != "Deskripsi berbeda." {
		t.Fatalf("officer include_low: %+v", l)
	}
	if status, _ := lostStatus(t, owner.Client, lost.Code); status != "reported" {
		t.Fatalf("a low candidate does not change the status: %s", status)
	}
}

// US-18 and model-lifecycle 2: embeddings carry their version, a new revision is refused, an outage retries
func TestEmbeddingsWithTheInferenceService(t *testing.T) {
	ml := testenv.NewFakeML(t)
	e := isolated(t, testenv.WithML(ml.URL))
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	officer := e.User(access.Officer, e.Post("pos-perpustakaan"))
	eng := e.Services.Engine
	embedImage, embedText, analyze := &matching.EmbedImageWorker{E: eng}, &matching.EmbedTextWorker{E: eng}, &matching.AnalyzeImageWorker{E: eng}
	work := func(results []error) {
		t.Helper()
		for _, err := range results {
			if err != nil {
				t.Fatalf("job: %v", err)
			}
		}
	}

	lost := fileLost(t, e, owner, "ml-lost-0001", nil)
	found := fileFound(t, e, finder, "ml-found-0001", nil)
	// match.report waits while embeddings are pending
	results := testenv.Drain(e, &matching.MatchReportWorker{E: eng})
	var snooze *river.JobSnoozeError
	if len(results) != 2 || !errors.As(results[0], &snooze) {
		t.Fatalf("scoring waits for embeddings: %v", results)
	}
	work(testenv.Drain(e, embedImage))
	work(testenv.Drain(e, analyze))
	work(testenv.Drain(e, embedText))
	var versions, images, texts int
	ctx := context.Background()
	_ = e.Pool.QueryRow(ctx, "SELECT count(*) FROM model_versions WHERE status = 'active' AND component IN ('image_encoder', 'text_encoder', 'xmodal_text_encoder')").Scan(&versions)
	_ = e.Pool.QueryRow(ctx, "SELECT count(*) FROM image_embeddings WHERE report_id = $1", found.ID).Scan(&images)
	_ = e.Pool.QueryRow(ctx, "SELECT count(*) FROM text_embeddings_e5 WHERE report_id = ANY($1)", []string{lost.ID, found.ID}).Scan(&texts)
	if versions != 3 || images != 1 || texts != 2 {
		t.Fatalf("versions %d, image embeddings %d, text embeddings %d", versions, images, texts)
	}

	if _, err := e.Pool.Exec(ctx, "UPDATE river_job SET scheduled_at = now() WHERE kind = 'match.report' AND state = 'scheduled'"); err != nil {
		t.Fatal(err)
	}
	score(t, e)
	// the fake vectors carry no meaning, so the pair scores low and only include_low shows it
	l := candidates(t, officer.Client, "/api/v1/lost-reports/"+lost.Code+"/matches?include_low=true")
	if len(l.Items) != 1 || l.Items[0].Explanation.VisualMode != "cross_modal" || l.Items[0].Scores.Visual == nil || l.Items[0].Scores.Semantic == nil {
		t.Fatalf("a lost report without photos is compared through its text: %+v", l)
	}

	// an unchanged text is not embedded again
	calls := ml.Calls["/v1/embed/text"]
	if res := officer.Client.JSON(http.MethodPost, "/api/v1/reports/lost/"+lost.Code+"/rematch", nil); res.Status != http.StatusAccepted {
		t.Fatalf("rematch: %d %s", res.Status, res.Body)
	}
	if r := testenv.Drain(e, embedText); len(r) != 1 || r[0] != nil {
		t.Fatalf("rematch queues one text job: %v", r)
	}
	if ml.Calls["/v1/embed/text"] != calls {
		t.Fatalf("unchanged text embedded again")
	}

	// an outage is retried, a missing endpoint and a new revision are cancelled
	ml.Fail(http.StatusServiceUnavailable)
	fileFound(t, e, finder, "ml-found-0002", nil)
	var cancel *river.JobCancelError
	if r := testenv.Drain(e, embedImage); len(r) != 1 || r[0] == nil || errors.As(r[0], &cancel) {
		t.Fatalf("outage: %v", r)
	}
	ml.Fail(0)
	work(testenv.Drain(e, embedImage))
	ml.Fail(http.StatusNotImplemented)
	fileFound(t, e, finder, "ml-found-0003", nil)
	if r := testenv.Drain(e, embedImage); len(r) != 1 || !errors.As(r[0], &cancel) {
		t.Fatalf("unavailable: %v", r)
	}
	ml.Fail(0)
	ml.Revision("r2")
	fileFound(t, e, finder, "ml-found-0004", nil)
	if r := testenv.Drain(e, embedImage); len(r) != 1 || !errors.As(r[0], &cancel) {
		t.Fatalf("new revision: %v", r)
	}
	var candidatesN int
	_ = e.Pool.QueryRow(ctx, "SELECT count(*) FROM model_versions WHERE component = 'image_encoder' AND status = 'candidate'").Scan(&candidatesN)
	if candidatesN != 1 {
		t.Fatalf("the new revision is recorded as a candidate: %d", candidatesN)
	}
}

// matching-engine 6.2: a card that names the owner lifts the text signal, and the owner sees only the blurred photo
func TestIdentityCardNamingTheOwner(t *testing.T) {
	ml := testenv.NewFakeML(t)
	e := isolated(t, testenv.WithML(ml.URL))
	owner := e.User(access.Member)
	finder := e.User(access.Member)
	eng := e.Services.Engine
	ml.ShowText(strings.ToUpper(owner.FullName) + " NIM 24/543770/PA/23107")

	lost := fileLost(t, e, owner, "ktm-lost-0001", func(b map[string]any) {
		b["category_id"], b["item_name"], b["colours"] = e.Category("ktm").String(), "KTM", []string{"biru"}
		b["description"], b["private_details"] = "Kartu mahasiswa di dompet plastik.", "Ada stiker kecil di belakang"
	})
	library := e.Post("pos-perpustakaan")
	fileFound(t, e, finder, "ktm-found-0001", func(b map[string]any) {
		b["category_id"], b["item_name"], b["colours"] = e.Category("ktm").String(), "Kartu mahasiswa", []string{"putih"}
		b["custody"] = map[string]any{"mode": "handover", "intended_post_id": library.String()}
	})
	for _, w := range []func() []error{
		func() []error { return testenv.Drain(e, &matching.EmbedImageWorker{E: eng}) },
		func() []error { return testenv.Drain(e, &matching.AnalyzeImageWorker{E: eng}) },
		func() []error { return testenv.Drain(e, &matching.EmbedTextWorker{E: eng}) },
	} {
		for _, err := range w() {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	score(t, e)
	l := candidates(t, owner.Client, "/api/v1/lost-reports/"+lost.Code+"/matches")
	if len(l.Items) != 1 || l.Items[0].Explanation.Text.Reason != "Nama pada barang sesuai dengan namamu." {
		t.Fatalf("name hit: %+v", l)
	}
	if p := l.Items[0].Found.Photos; len(p) != 1 || p[0].Variant != "thumb_blur" {
		t.Fatalf("identity photos: %+v", p)
	}
	media := "/api/v1/media/" + l.Items[0].Found.Photos[0].MediaID.String()
	if res := owner.Client.Do(http.MethodGet, media+"?variant=display", nil, ""); res.Status != http.StatusNotFound {
		t.Fatalf("identity display: %d", res.Status)
	}
	if res := owner.Client.Do(http.MethodGet, media+"?variant=thumb_blur", nil, ""); res.Status != http.StatusOK {
		t.Fatalf("identity thumb_blur: %d", res.Status)
	}
	raw := string(owner.Client.JSON(http.MethodGet, "/api/v1/lost-reports/"+lost.Code+"/matches", nil).Body)
	if strings.Contains(raw, "543770") || strings.Contains(raw, "NIM") {
		t.Fatalf("visible text leaks to the owner: %s", raw)
	}
}
