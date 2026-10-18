package matching

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/platform/db"
	"github.com/pisondev/simpanugm/api/internal/privacy"
)

const (
	// section 10: match.report waits for pending embeddings at most this long, then scores with what exists
	embedWait   = 60 * time.Second
	embedSnooze = 10 * time.Second
	// the variant sent to ml, large enough for OCR and small enough for one request
	mlVariant = "display"
)

var errNoInference = errors.New("matching: no inference service configured")

func validType(t string) bool { return t == TypeLost || t == TypeFound }

// cancels what retrying cannot fix, everything else goes back to River with backoff
func mlFailure(err error) error {
	switch {
	case errors.Is(err, mlclient.ErrUnavailable), errors.Is(err, ErrVersionMismatch):
		return river.JobCancel(err)
	default:
		return err
	}
}

func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

// the display variant of one photo that still belongs to the report, false when it was removed meanwhile
func (e *Engine) mediaBytes(ctx context.Context, mediaID uuid.UUID, reportType string, reportID uuid.UUID) ([]byte, bool, error) {
	row, err := e.q.GetMediaForEmbedding(ctx, mediaID)
	if db.IsNoRows(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	owner := row.LostReportID
	if reportType == TypeFound {
		owner = row.FoundReportID
	}
	if row.State != "ready" || owner == nil || *owner != reportID {
		return nil, false, nil
	}
	var keys map[string]string
	if err := json.Unmarshal(row.StorageKeys, &keys); err != nil || keys[mlVariant] == "" {
		return nil, false, fmt.Errorf("matching: media %s has no %s variant", mediaID, mlVariant)
	}
	body, _, err := e.store.Get(ctx, keys[mlVariant])
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = body.Close() }()
	b, err := io.ReadAll(body)
	return b, err == nil, err
}

type EmbedImageWorker struct {
	river.WorkerDefaults[jobargs.EmbedImage]
	E *Engine
}

func (w *EmbedImageWorker) Work(ctx context.Context, job *river.Job[jobargs.EmbedImage]) error {
	e, a := w.E, job.Args
	if e.ml == nil {
		return river.JobCancel(errNoInference)
	}
	if !validType(a.ReportType) {
		return river.JobCancel(fmt.Errorf("matching: report type %q", a.ReportType))
	}
	img, ok, err := e.mediaBytes(ctx, a.MediaID, a.ReportType, a.ReportID)
	if err != nil || !ok {
		return err
	}
	out, err := e.ml.EmbedImage(ctx, [][]byte{img})
	if err != nil {
		return mlFailure(err)
	}
	// registered outside the write so a candidate version stays recorded for an administrator even when it is refused
	vid, err := registerEncoder(ctx, e.q, ComponentImage, out.Model, out.Revision, e.clock.Now())
	if err != nil {
		return mlFailure(err)
	}
	return e.q.UpsertImageEmbedding(ctx, sqlc.UpsertImageEmbeddingParams{
		MediaID: a.MediaID, ModelVersionID: vid, ReportType: a.ReportType, ReportID: a.ReportID,
		Embedding: vectorLiteral(out.Embeddings[0]), CreatedAt: e.clock.Now(),
	})
}

func (e *Engine) synonyms(ctx context.Context) (map[string]string, error) {
	rows, err := e.q.ListSynonyms(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[strings.ToLower(r.Term)] = strings.ToLower(r.Expansion)
	}
	return out, nil
}

func textInput(r *report) TextInput {
	return TextInput{
		ItemName: r.ItemName, CategoryName: r.CategoryName, ParentName: r.ParentName, Colours: r.Colours, Brand: r.Brand,
		Description: r.Description, Private: r.Private, VisibleText: r.VisibleText,
	}
}

type EmbedTextWorker struct {
	river.WorkerDefaults[jobargs.EmbedText]
	E *Engine
}

func (w *EmbedTextWorker) Work(ctx context.Context, job *river.Job[jobargs.EmbedText]) error {
	e, a := w.E, job.Args
	if e.ml == nil {
		return river.JobCancel(errNoInference)
	}
	if !validType(a.ReportType) {
		return river.JobCancel(fmt.Errorf("matching: report type %q", a.ReportType))
	}
	r, err := e.load(ctx, e.q, a.ReportType, a.ReportID)
	if err != nil || r == nil || !r.Open {
		return err
	}
	syn, err := e.synonyms(ctx)
	if err != nil {
		return err
	}
	in := textInput(r)
	e5Text, clipText := E5Text(in, syn), ClipText(in, syn)
	e5Hash, clipHash := textHash(e5Text), textHash(clipText)
	// the CLIP text is built from a subset of the same fields, so an unchanged e5 text means nothing to do
	if textID, err := activeEncoder(ctx, e.q, ComponentText); err == nil && textID != uuid.Nil {
		if xID, err := activeEncoder(ctx, e.q, ComponentXModal); err == nil && xID != uuid.Nil {
			if stored, err := e.q.TextHashE5(ctx, sqlc.TextHashE5Params{ReportType: a.ReportType, ReportID: a.ReportID, ModelVersionID: textID}); err == nil && string(stored) == string(e5Hash) {
				return nil
			}
		}
	}
	e5, err := e.ml.EmbedText(ctx, []string{e5Text}, []string{"e5"})
	if err != nil {
		return mlFailure(err)
	}
	clip, err := e.ml.EmbedText(ctx, []string{clipText}, []string{"clip_text"})
	if err != nil {
		return mlFailure(err)
	}
	now := e.clock.Now()
	textID, err := registerEncoder(ctx, e.q, ComponentText, e5["e5"].Model, e5["e5"].Revision, now)
	if err != nil {
		return mlFailure(err)
	}
	xID, err := registerEncoder(ctx, e.q, ComponentXModal, clip["clip_text"].Model, clip["clip_text"].Revision, now)
	if err != nil {
		return mlFailure(err)
	}
	return db.InTx(ctx, e.pool, func(tx pgx.Tx) error {
		q := e.q.WithTx(tx)
		if err := q.UpsertTextE5(ctx, sqlc.UpsertTextE5Params{
			ReportType: a.ReportType, ReportID: a.ReportID, ModelVersionID: textID, TextSha256: e5Hash, CreatedAt: now,
			Embedding: vectorLiteral(e5["e5"].Embeddings[0]),
		}); err != nil {
			return err
		}
		return q.UpsertTextClip(ctx, sqlc.UpsertTextClipParams{
			ReportType: a.ReportType, ReportID: a.ReportID, ModelVersionID: xID, TextSha256: clipHash, CreatedAt: now,
			Embedding: vectorLiteral(clip["clip_text"].Embeddings[0]),
		})
	})
}

type AnalyzeImageWorker struct {
	river.WorkerDefaults[jobargs.AnalyzeImage]
	E *Engine
}

// the categories and their prompts, with a digest so ml can cache its prompt embeddings
func (e *Engine) promptSet(ctx context.Context) ([]mlclient.CategoryPrompts, string, error) {
	rows, err := e.q.CategoryPromptSet(ctx)
	if err != nil {
		return nil, "", err
	}
	set := make([]mlclient.CategoryPrompts, 0, len(rows))
	for _, r := range rows {
		if len(r.ClipPrompts) > 0 {
			set = append(set, mlclient.CategoryPrompts{Slug: r.Slug, Prompts: r.ClipPrompts})
		}
	}
	b, _ := json.Marshal(set)
	sum := sha256.Sum256(b)
	return set, hex.EncodeToString(sum[:]), nil
}

func (w *AnalyzeImageWorker) Work(ctx context.Context, job *river.Job[jobargs.AnalyzeImage]) error {
	e, a := w.E, job.Args
	if e.ml == nil {
		return river.JobCancel(errNoInference)
	}
	if !validType(a.ReportType) {
		return river.JobCancel(fmt.Errorf("matching: report type %q", a.ReportType))
	}
	img, ok, err := e.mediaBytes(ctx, a.MediaID, a.ReportType, a.ReportID)
	if err != nil || !ok {
		return err
	}
	set, hash, err := e.promptSet(ctx)
	if err != nil {
		return err
	}
	out, err := e.ml.AnalyzeImage(ctx, img, set, hash)
	if err != nil {
		return mlFailure(err)
	}
	versions, _ := json.Marshal(out.Versions)
	cats, _ := json.Marshal(out.Categories)
	colours, _ := json.Marshal(out.Colours)
	brands, _ := json.Marshal(out.Brands)
	var visible *string
	// ml masks long digit runs already, masking again keeps a card number out even when it did not
	if t := strings.TrimSpace(privacy.MaskNumbers(out.Text.Value)); t != "" {
		visible = &t
	}
	return db.InTx(ctx, e.pool, func(tx pgx.Tx) error {
		q := e.q.WithTx(tx)
		if err := q.UpsertImageAttributes(ctx, sqlc.UpsertImageAttributesParams{
			MediaID: a.MediaID, Versions: versions, CategoryScores: nonNull(cats), Colours: nonNull(colours), VisibleText: visible,
			Brands: nonNull(brands), CreatedAt: e.clock.Now(),
		}); err != nil {
			return err
		}
		// the text a photo shows joins the report's e5 text
		_, err := e.jobs.InsertTx(ctx, tx, jobargs.EmbedText{ReportType: a.ReportType, ReportID: a.ReportID}, nil)
		return err
	})
}

func nonNull(b []byte) []byte {
	if string(b) == "null" {
		return []byte("[]")
	}
	return b
}

const pendingEmbeddingsSQL = `SELECT EXISTS (
    SELECT 1 FROM river_job
    WHERE kind = ANY($1) AND state IN ('available', 'pending', 'running', 'scheduled', 'retryable')
      AND args->>'report_type' = $2 AND args->>'report_id' = $3
)`

func (e *Engine) pendingEmbeddings(ctx context.Context, reportType string, reportID uuid.UUID) (bool, error) {
	kinds := []string{jobargs.EmbedImage{}.Kind(), jobargs.AnalyzeImage{}.Kind(), jobargs.EmbedText{}.Kind()}
	var pending bool
	err := e.pool.QueryRow(ctx, pendingEmbeddingsSQL, kinds, reportType, reportID.String()).Scan(&pending)
	return pending, err
}

type MatchReportWorker struct {
	river.WorkerDefaults[jobargs.MatchReport]
	E *Engine
}

func (w *MatchReportWorker) Work(ctx context.Context, job *river.Job[jobargs.MatchReport]) error {
	e, a := w.E, job.Args
	if !validType(a.ReportType) {
		return river.JobCancel(fmt.Errorf("matching: report type %q", a.ReportType))
	}
	if e.ml != nil && time.Since(job.CreatedAt) < embedWait {
		pending, err := e.pendingEmbeddings(ctx, a.ReportType, a.ReportID)
		if err != nil {
			return err
		}
		if pending {
			return river.JobSnooze(embedSnooze)
		}
	}
	err := e.ScoreReport(ctx, a.ReportType, a.ReportID)
	if errors.Is(err, ErrChangedMeanwhile) {
		return river.JobSnooze(time.Second)
	}
	return err
}

type MatchNotifyWorker struct {
	river.WorkerDefaults[jobargs.MatchNotify]
	E *Engine
}

// BR-NTF-01: one match notification per lost report per window, whatever arrived inside the window is folded into the next
func (w *MatchNotifyWorker) Work(ctx context.Context, job *river.Job[jobargs.MatchNotify]) error {
	e := w.E
	lost, err := e.load(ctx, e.q, TypeLost, job.Args.LostReportID)
	if err != nil || lost == nil || !lost.Open || lost.ReporterID == nil {
		return err
	}
	owner := *lost.ReporterID
	now := e.clock.Now()
	last, err := e.q.LastMatchNotification(ctx, sqlc.LastMatchNotificationParams{RecipientID: owner, EntityID: &lost.ID})
	if err != nil {
		return err
	}
	window := time.Duration(e.settings.Int(ctx, "notify.match_throttle_hours")) * time.Hour
	if next := last.Add(window); now.Before(next) {
		return river.JobSnooze(next.Sub(now))
	}
	return db.InTx(ctx, e.pool, func(tx pgx.Tx) error {
		q := e.q.WithTx(tx)
		rows, err := q.UnnotifiedCandidates(ctx, lost.ID)
		if err != nil || len(rows) == 0 {
			return err
		}
		ids := make([]uuid.UUID, 0, len(rows))
		high := false
		for _, r := range rows {
			ids = append(ids, r.ID)
			high = high || r.Tier == "high"
		}
		msg := notify.Message{
			Event: notify.MatchMediumForOwner, To: owner, Link: "/laporan/hilang/" + lost.Code + "#kandidat",
			Data:       map[string]string{"count": strconv.Itoa(len(rows)), "item_name": lost.ItemName},
			EntityType: "lost_report", EntityID: &lost.ID,
		}
		if high {
			msg.Event, msg.Data = notify.MatchHighForOwner, map[string]string{"lost_code": lost.Code}
		}
		if err := e.notifier.Send(ctx, tx, msg); err != nil {
			return err
		}
		return q.MarkNotified(ctx, ids)
	})
}

type MatchSweepWorker struct {
	river.WorkerDefaults[jobargs.MatchSweep]
	E *Engine
}

const sweepBatch = 500

// section 10: every open report is scored again, and photos or texts still missing an embedding are queued first
func (w *MatchSweepWorker) Work(ctx context.Context, _ *river.Job[jobargs.MatchSweep]) error {
	e := w.E
	reports, err := e.q.OpenReportsForSweep(ctx)
	if err != nil {
		return err
	}
	imageID := uuid.Nil
	if e.ml != nil {
		if imageID, err = activeEncoder(ctx, e.q, ComponentImage); err != nil {
			return err
		}
	}
	var batch []river.InsertManyParams
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := e.jobs.InsertMany(ctx, batch)
		batch = batch[:0]
		return err
	}
	for _, r := range reports {
		if e.ml != nil {
			id := r.ID
			missing, err := e.q.MediaMissingEmbedding(ctx, sqlc.MediaMissingEmbeddingParams{ReportType: r.ReportType, ReportID: &id, VersionID: imageID})
			if err != nil {
				return err
			}
			for _, m := range missing {
				batch = append(batch, river.InsertManyParams{Args: jobargs.EmbedImage{MediaID: m, ReportType: r.ReportType, ReportID: r.ID}})
			}
			batch = append(batch, river.InsertManyParams{Args: jobargs.EmbedText{ReportType: r.ReportType, ReportID: r.ID}})
		}
		batch = append(batch, river.InsertManyParams{Args: jobargs.MatchReport{ReportType: r.ReportType, ReportID: r.ID}})
		if len(batch) >= sweepBatch {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func Periodic() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(notify.DailyAt{Hour: 2}, func() (river.JobArgs, *river.InsertOpts) {
			return jobargs.MatchSweep{}, nil
		}, nil),
	}
}

func AddWorkers(workers *river.Workers, e *Engine) {
	river.AddWorker(workers, &EmbedImageWorker{E: e})
	river.AddWorker(workers, &EmbedTextWorker{E: e})
	river.AddWorker(workers, &AnalyzeImageWorker{E: e})
	river.AddWorker(workers, &MatchReportWorker{E: e})
	river.AddWorker(workers, &MatchNotifyWorker{E: e})
	river.AddWorker(workers, &MatchSweepWorker{E: e})
}
