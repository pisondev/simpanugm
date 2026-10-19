package demo

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/analytics"
	"github.com/pisondev/simpanugm/api/internal/auth"
	"github.com/pisondev/simpanugm/api/internal/claims"
	"github.com/pisondev/simpanugm/api/internal/custody"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/masterdata"
	"github.com/pisondev/simpanugm/api/internal/matching"
	"github.com/pisondev/simpanugm/api/internal/matching/jobargs"
	"github.com/pisondev/simpanugm/api/internal/media"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/notify"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/config"
	"github.com/pisondev/simpanugm/api/internal/platform/ids"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
	"github.com/pisondev/simpanugm/api/internal/platform/storage"
	"github.com/pisondev/simpanugm/api/internal/reports"
	"github.com/pisondev/simpanugm/api/migrations"
)

func sampleJPEG(seed int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x + seed*29) % 256),
				G: uint8((y + seed*53) % 256),
				B: uint8((x*y/10 + seed*71) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return buf.Bytes()
}

//nolint:gosec // standard deterministic password for demo mode
const DemoPassword = "kata sandi panjang 2026"

type Deps struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	SQLDB    *sql.DB
	Clock    clock.Clock
	Store    storage.Store
	ML       *mlclient.Client
	Jobs     *river.Client[pgx.Tx]
	SeedDirs []string
	Reset    bool
}

type Result struct {
	UsersCount         int
	LostReportsCount   int
	FoundReportsCount  int
	MatchesCount       int
	ClaimsCount        int
	DuplicatesCount    int
	ClustersCount      int
	NotificationsCount int
}

func principalFor(u sqlc.User, postIDs []uuid.UUID) *access.Principal {
	return &access.Principal{
		UserID:          u.ID,
		Email:           u.Email,
		FullName:        u.FullName,
		Role:            access.Role(u.Role),
		Status:          u.Status,
		EmailVerified:   u.EmailVerifiedAt != nil,
		ClaimRestricted: u.ClaimRestrictedAt != nil,
		PostIDs:         postIDs,
	}
}

func Run(ctx context.Context, deps Deps) (*Result, error) {
	if deps.Config.IsProduction() {
		return nil, errors.New("simpan demo is refused in production")
	}

	if deps.Reset && deps.SQLDB != nil {
		slog.Info("resetting demo database with goose migrations")
		goose.SetBaseFS(migrations.FS)
		goose.SetLogger(goose.NopLogger())
		_ = goose.SetDialect("postgres")
		if err := goose.DownToContext(ctx, deps.SQLDB, ".", 0); err != nil {
			return nil, fmt.Errorf("reset database (goose down): %w", err)
		}
		if err := goose.UpContext(ctx, deps.SQLDB, "."); err != nil {
			return nil, fmt.Errorf("reset database (goose up): %w", err)
		}
		migrator, err := rivermigrate.New(riverpgxv5.New(deps.Pool), nil)
		if err != nil {
			return nil, fmt.Errorf("river migrator: %w", err)
		}
		if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			return nil, fmt.Errorf("river migrations: %w", err)
		}
		_, _ = deps.Pool.Exec(ctx, "TRUNCATE TABLE river_job, river_client, river_queue CASCADE;")
	}

	slog.Info("applying master data seeds for demo")
	seedDirs := deps.SeedDirs
	if len(seedDirs) == 0 {
		seedDirs = []string{"data/seed"}
	}
	set, err := masterdata.LoadSeed(seedDirs...)
	if err != nil {
		return nil, fmt.Errorf("load masterdata seed: %w", err)
	}
	if _, err := masterdata.ApplySeed(ctx, deps.Pool, set); err != nil {
		return nil, fmt.Errorf("apply masterdata seed: %w", err)
	}
	if err := matching.EnsureDefaultFusion(ctx, sqlc.New(deps.Pool), deps.Clock.Now()); err != nil {
		return nil, fmt.Errorf("ensure default fusion: %w", err)
	}

	postsBySlug := make(map[string]uuid.UUID)
	allPostIDs := make([]uuid.UUID, 0)
	postRows, err := deps.Pool.Query(ctx, "SELECT id, slug FROM posts")
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	for postRows.Next() {
		var id uuid.UUID
		var slug string
		if err := postRows.Scan(&id, &slug); err == nil {
			postsBySlug[slug] = id
			allPostIDs = append(allPostIDs, id)
		}
	}
	postRows.Close()

	locationsBySlug := make(map[string]uuid.UUID)
	locRows, err := deps.Pool.Query(ctx, "SELECT id, slug FROM locations")
	if err != nil {
		return nil, fmt.Errorf("query locations: %w", err)
	}
	for locRows.Next() {
		var id uuid.UUID
		var slug string
		if err := locRows.Scan(&id, &slug); err == nil {
			locationsBySlug[slug] = id
		}
	}
	locRows.Close()

	categoriesBySlug := make(map[string]uuid.UUID)
	catRows, err := deps.Pool.Query(ctx, "SELECT id, slug FROM categories WHERE parent_id IS NOT NULL")
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	for catRows.Next() {
		var id uuid.UUID
		var slug string
		if err := catRows.Scan(&id, &slug); err == nil {
			categoriesBySlug[slug] = id
		}
	}
	catRows.Close()

	st := settings.New(deps.Pool, sqlc.New(deps.Pool), deps.Clock.Now)
	mailer := notify.NewMailer(deps.Jobs, strings.TrimRight(deps.Config.PublicBaseURL, "/"))
	authSvc := auth.NewService(deps.Pool, deps.Clock, mailer, st)

	slog.Info("creating demo accounts")
	userMap := make(map[string]sqlc.User)
	for _, acc := range DemoAccounts {
		var posts []uuid.UUID
		if acc.AllPosts {
			posts = allPostIDs
		} else if acc.PostSlug != "" {
			if pid, ok := postsBySlug[acc.PostSlug]; ok {
				posts = []uuid.UUID{pid}
			}
		}
		u, err := authSvc.CreateActiveUser(ctx, auth.ActiveUser{
			Email:    acc.Email,
			Name:     acc.Name,
			Role:     access.Role(acc.Role),
			Password: DemoPassword,
			PostIDs:  posts,
		})
		if err != nil {
			return nil, fmt.Errorf("create demo user %s: %w", acc.Email, err)
		}
		userMap[acc.Email] = u
	}

	notifier := notify.NewNotifier(deps.Jobs, deps.Config.PublicBaseURL, st, deps.Clock)
	med := media.NewService(deps.Pool, deps.Store, deps.Clock)
	md := masterdata.NewService(deps.Pool, deps.Clock, st)
	cs := custody.NewService(deps.Pool, deps.Clock, st, notifier)
	enq := &jobargs.Enqueuer{Jobs: deps.Jobs, Inference: deps.ML != nil}
	repSvc := reports.NewService(deps.Pool, deps.Clock, med, md, cs, st, notifier, enq)
	engine := matching.NewEngine(deps.Pool, deps.Clock, deps.ML, deps.Store, notifier, st, deps.Jobs)
	claimsSvc := claims.NewService(deps.Pool, deps.Clock, st, notifier, med, md, cs, repSvc, deps.Config.HandoverPepper)
	analyticsSvc := analytics.NewService(deps.Pool, deps.Clock, deps.ML, st)

	slog.Info("creating demo lost and found reports")
	allLost := make([]DemoLostReport, 0, len(HeroLostReports)+len(BackgroundLostReports))
	allLost = append(allLost, HeroLostReports...)
	allLost = append(allLost, BackgroundLostReports...)

	allFound := make([]DemoFoundReport, 0, len(HeroFoundReports)+len(BackgroundFoundReports))
	allFound = append(allFound, HeroFoundReports...)
	allFound = append(allFound, BackgroundFoundReports...)

	lostMap := make(map[string]openapi.LostReport)
	foundMap := make(map[string]openapi.FoundReport)
	now := deps.Clock.Now()

	for idx, r := range allLost {
		catID := categoriesBySlug[r.CategorySlug]
		locID := locationsBySlug[r.LocationSlug]
		lostFrom := now.Add(-time.Duration(r.DaysAgo*24) * time.Hour)
		lostUntil := lostFrom.Add(4 * time.Hour)

		var outcomeBytes []byte
		if r.AssistOutcome != nil {
			outcomeBytes, _ = json.Marshal(r.AssistOutcome)
		}

		in := reports.LostInput{
			CategoryID:    catID,
			ItemName:      r.ItemName,
			Description:   r.Description,
			Colours:       r.Colours,
			LostFrom:      lostFrom,
			LostUntil:     lostUntil,
			Places:        []reports.Place{{LocationID: &locID}},
			MediaIDs:      []uuid.UUID{},
			AssistOutcome: outcomeBytes,
		}
		if r.Brand != "" {
			in.Brand = &r.Brand
		}
		if r.PrivateDetails != "" {
			in.PrivateDetails = &r.PrivateDetails
		}

		var p *access.Principal
		if strings.HasPrefix(r.Key, "bg-lost-") {
			officerUser := userMap["petugas@demo.simpanugm.my.id"]
			p = principalFor(officerUser, allPostIDs)
			visitorName := fmt.Sprintf("Pengunjung Kampus %02d", idx+1)
			in.ProxyName = &visitorName
		} else {
			u, ok := userMap[r.OwnerEmail]
			if !ok {
				u = userMap["anggota@demo.simpanugm.my.id"]
			}
			p = principalFor(u, nil)
		}

		idemKey := fmt.Sprintf("demo-lost-%d-%s", idx, r.Key)
		var existingLostID *uuid.UUID
		var existingLostCode string
		err := deps.Pool.QueryRow(ctx, `
			SELECT r.id, r.code FROM idempotency_keys k
			JOIN lost_reports r ON r.id = k.resource_id
			WHERE k.user_id = $1 AND k.key = $2 AND k.endpoint = 'createLostReport'
		`, p.UserID, idemKey).Scan(&existingLostID, &existingLostCode)
		if err == nil && existingLostID != nil {
			created, err := repSvc.GetLost(ctx, p, existingLostCode)
			if err == nil {
				lostMap[r.Key] = created
				continue
			}
		}

		created, err := repSvc.CreateLost(ctx, p, idemKey, in)
		if err != nil {
			return nil, fmt.Errorf("create lost report %s: %w", r.Key, err)
		}
		lostMap[r.Key] = created
	}

	for idx, r := range allFound {
		catID := categoriesBySlug[r.CategorySlug]
		locID := locationsBySlug[r.LocationSlug]
		postID := postsBySlug[r.PostSlug]
		if postID == uuid.Nil && len(allPostIDs) > 0 {
			postID = allPostIDs[0]
		}
		foundAt := now.Add(-time.Duration(r.DaysAgo*24) * time.Hour)

		var outcomeBytes []byte
		if r.AssistOutcome != nil {
			outcomeBytes, _ = json.Marshal(r.AssistOutcome)
		}

		var p *access.Principal
		var proxyInput *reports.ProxyFound
		if strings.HasPrefix(r.Key, "bg-found-") {
			officerUser := userMap["petugas@demo.simpanugm.my.id"]
			p = principalFor(officerUser, allPostIDs)
			finderName := fmt.Sprintf("Penemu Kampus %02d", idx+1)
			slot := r.StorageSlot
			if slot == "" {
				slot = "Rak B1"
			}
			proxyInput = &reports.ProxyFound{
				FinderName:  &finderName,
				PostID:      postID,
				StorageSlot: slot,
			}
		} else {
			u, ok := userMap[r.FinderEmail]
			if !ok {
				u = userMap["anggota@demo.simpanugm.my.id"]
			}
			p = principalFor(u, nil)
		}

		idemKey := fmt.Sprintf("demo-found-%d-%s", idx, r.Key)
		var existingFoundID *uuid.UUID
		var existingFoundCode string
		err := deps.Pool.QueryRow(ctx, `
			SELECT r.id, r.code FROM idempotency_keys k
			JOIN found_reports r ON r.id = k.resource_id
			WHERE k.user_id = $1 AND k.key = $2 AND k.endpoint = 'createFoundReport'
		`, p.UserID, idemKey).Scan(&existingFoundID, &existingFoundCode)
		if err == nil && existingFoundID != nil {
			created, err := repSvc.GetFound(ctx, p, existingFoundCode)
			if err == nil {
				foundMap[r.Key] = created
				continue
			}
		}

		m, err := med.Upload(ctx, p, "found_report", bytes.NewReader(sampleJPEG(idx+1)))
		if err != nil {
			return nil, fmt.Errorf("upload demo found photo %s: %w", r.Key, err)
		}

		custodyMode := "handover"
		if r.CustodyMode == "keep" {
			custodyMode = "keep"
		}

		in := reports.FoundInput{
			CategoryID:      catID,
			ItemName:        r.ItemName,
			Description:     r.Description,
			Colours:         r.Colours,
			FoundAt:         foundAt,
			FoundLocationID: locID,
			MediaIDs:        []uuid.UUID{m.ID},
			CustodyMode:     custodyMode,
			IntendedPostID:  &postID,
			Proxy:           proxyInput,
			AssistOutcome:   outcomeBytes,
		}
		if r.Brand != "" {
			in.Brand = &r.Brand
		}
		if r.HiddenDetails != "" {
			in.HiddenDetails = &r.HiddenDetails
		}

		created, err := repSvc.CreateFound(ctx, p, idemKey, in)
		if err != nil {
			return nil, fmt.Errorf("create found report %s: %w", r.Key, err)
		}
		foundMap[r.Key] = created

		if in.Proxy == nil && r.CustodyMode == "in_custody" {
			officerUser := userMap["petugas@demo.simpanugm.my.id"]
			pOfficer := principalFor(officerUser, allPostIDs)
			slot := r.StorageSlot
			if slot == "" {
				slot = "Rak A1"
			}
			_, _ = repSvc.Receive(ctx, pOfficer, created.Code, postID, slot, nil)
		}
	}

	slog.Info("computing text embeddings and scoring matches via matching engine")
	embedWorker := &matching.EmbedTextWorker{E: engine}
	for _, lr := range lostMap {
		_ = embedWorker.Work(ctx, &river.Job[jobargs.EmbedText]{Args: jobargs.EmbedText{ReportType: "lost", ReportID: lr.Id}})
	}
	for _, fr := range foundMap {
		_ = embedWorker.Work(ctx, &river.Job[jobargs.EmbedText]{Args: jobargs.EmbedText{ReportType: "found", ReportID: fr.Id}})
	}

	for _, lr := range lostMap {
		_ = engine.ScoreReport(ctx, "lost", lr.Id)
	}

	slog.Info("seeding demo claims across lifecycle statuses")
	officerUser := userMap["petugas@demo.simpanugm.my.id"]
	pOfficer := principalFor(officerUser, allPostIDs)
	claimsCount := 0

	for idx, c := range DemoClaims {
		lostRep, ok1 := lostMap[c.LostKey]
		foundRep, ok2 := foundMap[c.FoundKey]
		if !ok1 || !ok2 {
			continue
		}
		claimantUser := userMap[c.ClaimantEmail]
		pClaimant := principalFor(claimantUser, nil)

		var answers []claims.AnswerInput
		for k, v := range c.Answers {
			answers = append(answers, claims.AnswerInput{Key: k, Answer: v})
		}

		in := claims.CreateInput{
			FoundID:   foundRep.Id,
			LostID:    &lostRep.Id,
			Statement: c.Statement,
			Answers:   answers,
		}
		idemClaimKey := fmt.Sprintf("demo-claim-%d", idx)
		var existingClaimID *uuid.UUID
		err := deps.Pool.QueryRow(ctx, `
			SELECT c.id FROM idempotency_keys k
			JOIN claims c ON c.id = k.resource_id
			WHERE k.user_id = $1 AND k.key = $2 AND k.endpoint = 'createClaim'
		`, pClaimant.UserID, idemClaimKey).Scan(&existingClaimID)
		if err == nil && existingClaimID != nil {
			claimsCount++
			continue
		}

		claim, err := claimsSvc.Create(ctx, pClaimant, idemClaimKey, in)
		if err != nil {
			slog.Warn("create demo claim failed", "key", c.LostKey, "error", err)
			continue
		}
		claimsCount++

		switch c.TargetStatus {
		case "submitted":
			// Leave as submitted for officer to review
		case "info_requested":
			_, _ = claimsSvc.PickUp(ctx, pOfficer, claim.Code)
			reqMsg := c.RequestMessage
			if reqMsg == "" {
				reqMsg = "Mohon berikan informasi tambahan mengenai nomor seri atau ciri khusus barang."
			}
			_, _ = claimsSvc.RequestInfo(ctx, pOfficer, claim.Code, reqMsg)
		case "approved":
			_, _ = claimsSvc.PickUp(ctx, pOfficer, claim.Code)
			cl, _ := claimsSvc.Checks(ctx, pOfficer, claim.Code)
			var checks []claims.CheckResult
			for _, it := range cl.Items {
				checks = append(checks, claims.CheckResult{ItemKey: it.ItemKey, Result: "pass"})
			}
			_, _ = claimsSvc.PutChecks(ctx, pOfficer, claim.Code, checks)
			note := c.DecisionNote
			if note == "" {
				note = "Ciri khusus dan jawaban verifikasi sesuai."
			}
			_, _ = claimsSvc.Approve(ctx, pOfficer, claim.Code, "", &note)
		case "rejected":
			_, _ = claimsSvc.PickUp(ctx, pOfficer, claim.Code)
			reason := c.DecisionReason
			if reason == "" {
				reason = "detail_mismatch"
			}
			note := c.DecisionNote
			if note == "" {
				note = "Ciri khusus tidak sesuai dengan barang temuan di pos."
			}
			_, _ = claimsSvc.Reject(ctx, pOfficer, claim.Code, reason, &note)
		case "completed":
			_, _ = claimsSvc.PickUp(ctx, pOfficer, claim.Code)
			cl, _ := claimsSvc.Checks(ctx, pOfficer, claim.Code)
			var checks []claims.CheckResult
			for _, it := range cl.Items {
				checks = append(checks, claims.CheckResult{ItemKey: it.ItemKey, Result: "pass"})
			}
			_, _ = claimsSvc.PutChecks(ctx, pOfficer, claim.Code, checks)
			note := c.DecisionNote
			if note == "" {
				note = "Verifikasi lengkap, serah terima selesai."
			}
			approved, err := claimsSvc.Approve(ctx, pOfficer, claim.Code, "", &note)
			if err == nil {
				claimView, err := claimsSvc.Get(ctx, pClaimant, approved.Code)
				if err == nil && claimView.HandoverCode != nil {
					hInput := claims.HandoverInput{
						ClaimID:       approved.Id,
						Code:          *claimView.HandoverCode,
						IDType:        "ktm",
						IDLast4:       "3107",
						NameConfirmed: true,
					}
					_, err = claimsSvc.Handover(ctx, pOfficer, foundRep.Code, hInput)
					if err == nil {
						_, _ = repSvc.ConfirmReceipt(ctx, pClaimant, lostRep.Code)
					}
				}
			}
		}
	}

	duplicatesCount := 0
	if g1, ok1 := foundMap["dup-glasses-1"]; ok1 {
		if g2, ok2 := foundMap["dup-glasses-2"]; ok2 {
			tag, err := deps.Pool.Exec(ctx, `
				INSERT INTO duplicate_links (id, report_type, report_a, report_b, score, status, created_at)
				VALUES ($1, 'found', $2, $3, 0.8800, 'suggested', $4)
				ON CONFLICT DO NOTHING
			`, ids.New(), g1.Id, g2.Id, now)
			if err != nil {
				slog.Warn("insert duplicate link failed", "error", err)
			} else if tag.RowsAffected() > 0 {
				duplicatesCount++
			}
		}
	}
	var existingDuplicates int
	_ = deps.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM duplicate_links").Scan(&existingDuplicates)
	if duplicatesCount == 0 && existingDuplicates > 0 {
		duplicatesCount = existingDuplicates
	}

	clustersCount := 0
	adminUser := userMap["admin@demo.simpanugm.my.id"]
	clusterRun, err := analyticsSvc.CreateAiClusterRun(ctx, &adminUser.ID)
	if err == nil && clusterRun != nil {
		clustersCount = clusterRun.ClusterCount
	}

	slog.Info("inserting demo notifications")
	notifsCount := 0
	notificationsData := []struct {
		Email string
		Event string
		Title string
		Body  string
		Link  string
	}{
		{
			Email: "anggota.pemilik@demo.simpanugm.my.id",
			Event: "match.report",
			Title: "Kandidat barang temuan baru",
			Body:  "Ditemukan barang temuan yang sangat mirip dengan Dompet Kulit Lipat Eiger Hitam milikmu di GIK.",
			Link:  "/laporan",
		},
		{
			Email: "anggota.pemilik@demo.simpanugm.my.id",
			Event: "claim.approved",
			Title: "Klaimmu telah disetujui",
			Body:  "Klaim Payung Lipat Otomatis Hitam telah disetujui. Ambil barang di Pos Keamanan Perpustakaan Pusat dengan kode serah terima.",
			Link:  "/klaim",
		},
		{
			Email: "anggota.pemilik@demo.simpanugm.my.id",
			Event: "claim.info_requested",
			Title: "Permintaan informasi tambahan klaim",
			Body:  "Petugas meminta keterangan tambahan mengenai stiker di cover kalkulator Casio.",
			Link:  "/klaim",
		},
		{
			Email: "anggota.penemu@demo.simpanugm.my.id",
			Event: "found.hint",
			Title: "Kemungkinan pemilik sedang mencari barangmu",
			Body:  "Ada laporan kehilangan aktif yang sangat cocok dengan barang yang kamu temukan di GIK.",
			Link:  "/laporan",
		},
		{
			Email: "petugas.pos1@demo.simpanugm.my.id",
			Event: "claim.created",
			Title: "Pengajuan klaim baru",
			Body:  "Pengguna mengajukan klaim untuk Kunci Motor Honda Vario di Pos Klaster Saintek.",
			Link:  "/petugas/klaim",
		},
		{
			Email: "petugas@demo.simpanugm.my.id",
			Event: "custody.reminder",
			Title: "Papan tinjauan menunggu pemeriksaan",
			Body:  "Terdapat 5 laporan temuan baru siap ditinjau di Pos Keamanan Kampus.",
			Link:  "/petugas/papan",
		},
	}

	for _, n := range notificationsData {
		u := userMap[n.Email]
		_, err := deps.Pool.Exec(ctx, `
			INSERT INTO notifications (id, recipient_id, event, title, body, link, email_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'sent', $7)
		`, ids.New(), u.ID, n.Event, n.Title, n.Body, n.Link, now)
		if err == nil {
			notifsCount++
		}
	}

	slog.Info("refreshing analytics materialized views")
	_, _ = deps.Pool.Exec(ctx, `
		REFRESH MATERIALIZED VIEW mv_reports_daily;
		REFRESH MATERIALIZED VIEW mv_hour_weekday;
		REFRESH MATERIALIZED VIEW mv_return_times;
		REFRESH MATERIALIZED VIEW mv_match_outcomes;
	`)

	var matchCount int
	_ = deps.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM matches").Scan(&matchCount)

	result := &Result{
		UsersCount:         len(userMap),
		LostReportsCount:   len(lostMap),
		FoundReportsCount:  len(foundMap),
		MatchesCount:       matchCount,
		ClaimsCount:        claimsCount,
		DuplicatesCount:    duplicatesCount,
		ClustersCount:      clustersCount,
		NotificationsCount: notifsCount,
	}

	slog.Info("demo data seed complete",
		"users", result.UsersCount,
		"lost", result.LostReportsCount,
		"found", result.FoundReportsCount,
		"matches", result.MatchesCount,
		"claims", result.ClaimsCount,
		"duplicates", result.DuplicatesCount,
		"clusters", result.ClustersCount,
		"notifications", result.NotificationsCount,
	)

	return result, nil
}
