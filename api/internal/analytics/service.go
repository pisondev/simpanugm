package analytics

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/pisondev/simpanugm/api/internal/audit"
	"github.com/pisondev/simpanugm/api/internal/db/sqlc"
	"github.com/pisondev/simpanugm/api/internal/mlclient"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
	"github.com/pisondev/simpanugm/api/internal/platform/clock"
	"github.com/pisondev/simpanugm/api/internal/platform/settings"
)

type Service struct {
	pool     *pgxpool.Pool
	clock    clock.Clock
	ml       *mlclient.Client
	settings *settings.Store
	jobs     *river.Client[pgx.Tx]
}

func NewService(pool *pgxpool.Pool, clk clock.Clock, ml *mlclient.Client, st *settings.Store) *Service {
	return &Service{
		pool:     pool,
		clock:    clk,
		ml:       ml,
		settings: st,
	}
}

func (s *Service) WithJobs(jobs *river.Client[pgx.Tx]) *Service {
	s.jobs = jobs
	return s
}

func (s *Service) ListModelVersions(ctx context.Context, component *string) ([]openapi.ModelVersionItem, error) {
	query := `
		SELECT id, component, name, identity, config, eval_run_id, metrics, status, activated_at, activated_by, retired_at, created_at
		FROM model_versions
	`
	var args []any
	if component != nil && *component != "" {
		query += " WHERE component = $1"
		args = append(args, *component)
	}
	query += " ORDER BY component, CASE status WHEN 'active' THEN 1 WHEN 'candidate' THEN 2 ELSE 3 END, created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list model versions: %w", err)
	}
	defer rows.Close()

	var items []openapi.ModelVersionItem
	for rows.Next() {
		var it openapi.ModelVersionItem
		var id uuid.UUID
		var comp, name, status string
		var identityBytes, configBytes, metricsBytes []byte
		var evalRunID *string
		var activatedAt, retiredAt *time.Time
		var activatedBy *uuid.UUID
		var createdAt time.Time

		if err := rows.Scan(&id, &comp, &name, &identityBytes, &configBytes, &evalRunID, &metricsBytes, &status, &activatedAt, &activatedBy, &retiredAt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan model version: %w", err)
		}

		it.Id = id
		it.Component = comp
		it.Name = name
		it.Status = openapi.ModelVersionItemStatus(status)
		it.CreatedAt = createdAt
		it.EvalRunId = evalRunID
		it.ActivatedAt = activatedAt
		it.ActivatedBy = activatedBy
		it.RetiredAt = retiredAt

		if len(identityBytes) > 0 {
			var m map[string]any
			if err := json.Unmarshal(identityBytes, &m); err == nil {
				it.Identity = &m
			}
		}
		if len(configBytes) > 0 {
			var m map[string]any
			if err := json.Unmarshal(configBytes, &m); err == nil {
				it.Config = &m
			}
		}
		if len(metricsBytes) > 0 {
			var m map[string]any
			if err := json.Unmarshal(metricsBytes, &m); err == nil {
				it.Metrics = &m
			}
		}

		items = append(items, it)
	}
	if items == nil {
		items = []openapi.ModelVersionItem{}
	}
	return items, nil
}

func (s *Service) ActivateModelVersion(ctx context.Context, id uuid.UUID, actorID uuid.UUID) (*openapi.ModelVersionItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var component string
	err = tx.QueryRow(ctx, "SELECT component FROM model_versions WHERE id = $1", id).Scan(&component)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("query model version: %w", err)
	}

	now := s.clock.Now()
	// Retire currently active version of this component
	_, err = tx.Exec(ctx, `
		UPDATE model_versions
		SET status = 'retired', retired_at = $1
		WHERE component = $2 AND status = 'active'
	`, now, component)
	if err != nil {
		return nil, fmt.Errorf("retire active version: %w", err)
	}

	// Activate target version
	var it openapi.ModelVersionItem
	var comp, name, status string
	var identityBytes, configBytes, metricsBytes []byte
	var evalRunID *string
	var activatedAt, retiredAt *time.Time
	var activatedBy *uuid.UUID
	var createdAt time.Time

	err = tx.QueryRow(ctx, `
		UPDATE model_versions
		SET status = 'active', activated_at = $1, activated_by = $2, retired_at = NULL
		WHERE id = $3
		RETURNING id, component, name, identity, config, eval_run_id, metrics, status, activated_at, activated_by, retired_at, created_at
	`, now, actorID, id).Scan(
		&it.Id, &comp, &name, &identityBytes, &configBytes, &evalRunID, &metricsBytes, &status, &activatedAt, &activatedBy, &retiredAt, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("activate target version: %w", err)
	}

	qtx := sqlc.New(tx)
	if err := audit.Record(ctx, qtx, now, audit.Entry{
		Action:     "model_version.activate",
		EntityType: "model_version",
		EntityID:   id,
		Changes: map[string]any{
			"component": comp,
			"name":      name,
			"status":    "active",
		},
	}); err != nil {
		return nil, fmt.Errorf("audit activate model version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	it.Component = comp
	it.Name = name
	it.Status = openapi.ModelVersionItemStatus(status)
	it.CreatedAt = createdAt
	it.EvalRunId = evalRunID
	it.ActivatedAt = activatedAt
	it.ActivatedBy = activatedBy
	it.RetiredAt = retiredAt

	if len(identityBytes) > 0 {
		var m map[string]any
		_ = json.Unmarshal(identityBytes, &m)
		it.Identity = &m
	}
	if len(configBytes) > 0 {
		var m map[string]any
		_ = json.Unmarshal(configBytes, &m)
		it.Config = &m
	}
	if len(metricsBytes) > 0 {
		var m map[string]any
		_ = json.Unmarshal(metricsBytes, &m)
		it.Metrics = &m
	}

	return &it, nil
}

func (s *Service) GetAiMonitoring(ctx context.Context, from, to *time.Time, modelVersionID *uuid.UUID) (*openapi.AiMonitoringResponse, error) {
	// 1. Assist outcomes
	assistRows, err := s.pool.Query(ctx, "SELECT field_name, outcome, count FROM ai_monitoring_assist_outcomes")
	var assistOutcomes []openapi.AiMonitoringAssistOutcome
	if err == nil {
		defer assistRows.Close()
		for assistRows.Next() {
			var field, outcome string
			var count int64
			if err := assistRows.Scan(&field, &outcome, &count); err == nil {
				assistOutcomes = append(assistOutcomes, openapi.AiMonitoringAssistOutcome{
					Field:   field,
					Outcome: outcome,
					Count:   int(count),
				})
			}
		}
	}
	if assistOutcomes == nil {
		assistOutcomes = []openapi.AiMonitoringAssistOutcome{}
	}

	// 2. Dismiss rates
	dismissRows, err := s.pool.Query(ctx, "SELECT tier, total_matches, owner_dismissed_count, officer_dismissed_count, owner_dismiss_rate FROM ai_monitoring_dismiss_rates")
	var dismissRates []openapi.AiMonitoringDismissRate
	if err == nil {
		defer dismissRows.Close()
		for dismissRows.Next() {
			var tier string
			var total, ownerD, officerD int64
			var rate float64
			if err := dismissRows.Scan(&tier, &total, &ownerD, &officerD, &rate); err == nil {
				dismissRates = append(dismissRates, openapi.AiMonitoringDismissRate{
					Tier:             tier,
					Total:            int(total),
					OwnerDismissed:   int(ownerD),
					OfficerDismissed: int(officerD),
					DismissRate:      float32(rate),
				})
			}
		}
	}
	if dismissRates == nil {
		dismissRates = []openapi.AiMonitoringDismissRate{}
	}

	// 3. Confirmed ranks
	var confirmedRanks openapi.AiMonitoringConfirmedRanks
	var confTotal, r1, r5, r10 int64
	var rec1, rec5, rec10 float64
	err = s.pool.QueryRow(ctx, "SELECT confirmed_total, rank_1_count, rank_5_count, rank_10_count, recall_at_1, recall_at_5, recall_at_10 FROM ai_monitoring_confirmed_ranks").Scan(
		&confTotal, &r1, &r5, &r10, &rec1, &rec5, &rec10,
	)
	if err == nil {
		confirmedRanks = openapi.AiMonitoringConfirmedRanks{
			ConfirmedTotal: int(confTotal),
			Rank1Count:     int(r1),
			Rank5Count:     int(r5),
			Rank10Count:    int(r10),
			RecallAt1:      float32(rec1),
			RecallAt5:      float32(rec5),
			RecallAt10:     float32(rec10),
		}
	}

	// 4. Top-1 histogram
	histRows, err := s.pool.Query(ctx, "SELECT bucket_label, count, bucket_floor, bucket_ceil FROM ai_monitoring_top1_histogram ORDER BY bucket_floor")
	var top1Histogram []openapi.AiMonitoringHistogramBucket
	if err == nil {
		defer histRows.Close()
		for histRows.Next() {
			var label string
			var count int64
			var floor, ceil float64
			if err := histRows.Scan(&label, &count, &floor, &ceil); err == nil {
				top1Histogram = append(top1Histogram, openapi.AiMonitoringHistogramBucket{
					Bucket:      label,
					Count:       int(count),
					BucketFloor: float32(floor),
					BucketCeil:  float32(ceil),
				})
			}
		}
	}
	if top1Histogram == nil {
		top1Histogram = []openapi.AiMonitoringHistogramBucket{}
	}

	// 5. Score PSI against prior 4 weeks
	scorePSI, psiWarning := s.calculateScorePSI(ctx, from, to)

	// 6. Active versions
	activeVersions, _ := s.ListModelVersions(ctx, nil)
	var filteredActive []openapi.ModelVersionItem
	for _, v := range activeVersions {
		if v.Status == openapi.ModelVersionItemStatusActive {
			filteredActive = append(filteredActive, v)
		}
	}
	if filteredActive == nil {
		filteredActive = []openapi.ModelVersionItem{}
	}

	return &openapi.AiMonitoringResponse{
		From:           from,
		To:             to,
		AssistOutcomes: assistOutcomes,
		DismissRates:   dismissRates,
		ConfirmedRanks: confirmedRanks,
		Top1Histogram:  top1Histogram,
		ScorePsi:       float32(scorePSI),
		PsiWarning:     psiWarning,
		ActiveVersions: filteredActive,
	}, nil
}

func (s *Service) calculateScorePSI(ctx context.Context, from, to *time.Time) (float64, bool) {
	now := s.clock.Now()
	recentEnd := now
	if to != nil {
		recentEnd = *to
	}
	recentStart := recentEnd.Add(-28 * 24 * time.Hour)
	if from != nil {
		recentStart = *from
	}
	baselineStart := recentStart.Add(-28 * 24 * time.Hour)
	baselineEnd := recentStart

	getDistribution := func(start, end time.Time) ([]int, int) {
		counts := make([]int, 5)
		total := 0
		rows, err := s.pool.Query(ctx, `
			SELECT overall
			FROM matches
			WHERE rank_for_lost = 1 AND scored_at >= $1 AND scored_at < $2
		`, start, end)
		if err != nil {
			return counts, 0
		}
		defer rows.Close()

		for rows.Next() {
			var score float64
			if err := rows.Scan(&score); err == nil {
				total++
				switch {
				case score < 0.40:
					counts[0]++
				case score < 0.55:
					counts[1]++
				case score < 0.72:
					counts[2]++
				case score < 0.85:
					counts[3]++
				default:
					counts[4]++
				}
			}
		}
		return counts, total
	}

	countsP, totalP := getDistribution(recentStart, recentEnd)
	countsQ, totalQ := getDistribution(baselineStart, baselineEnd)

	if totalP == 0 || totalQ == 0 {
		return 0.0, false
	}

	eps := 0.0001
	var psi float64
	for i := 0; i < 5; i++ {
		p := (float64(countsP[i]) + eps) / (float64(totalP) + 5*eps)
		q := (float64(countsQ[i]) + eps) / (float64(totalQ) + 5*eps)
		psi += (p - q) * math.Log(p/q)
	}
	if math.IsNaN(psi) || math.IsInf(psi, 0) || psi < 0 {
		psi = 0.0
	}
	return psi, psi > 0.2
}

func (s *Service) CreateAiClusterRun(ctx context.Context, triggeredBy *uuid.UUID) (*openapi.ClusterRunSummary, error) {
	runID := uuid.New()
	startTime := s.clock.Now()
	since := startTime.Add(-30 * 24 * time.Hour)

	// Fetch open reports within 30 days
	type repItem struct {
		id      uuid.UUID
		repType string
	}
	var reports []repItem

	rows, err := s.pool.Query(ctx, `
		SELECT id, 'lost' AS rtype FROM lost_reports WHERE status IN ('open', 'registered') AND created_at >= $1
		UNION ALL
		SELECT id, 'found' AS rtype FROM found_reports WHERE custody_status IN ('with_finder', 'in_custody') AND found_at >= $1
	`, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r repItem
			if err := rows.Scan(&r.id, &r.repType); err == nil {
				reports = append(reports, r)
			}
		}
	}

	runParams := map[string]any{
		"min_cluster_size": 2,
		"min_samples":      1,
		"days":             30,
	}
	paramsBytes, _ := json.Marshal(runParams)
	versionsBytes := []byte(`{"algorithm": "hdbscan"}`)

	// If no reports or ML client is unavailable, record empty run
	if len(reports) < 2 || s.ml == nil {
		finishTime := s.clock.Now()
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO cluster_runs (id, params, versions, status, triggered_by, started_at, finished_at)
			VALUES ($1, $2, $3, 'done', $4, $5, $6)
		`, runID, paramsBytes, versionsBytes, triggeredBy, startTime, finishTime)

		return &openapi.ClusterRunSummary{
			Id:           runID,
			Status:       openapi.Done,
			StartedAt:    startTime,
			FinishedAt:   &finishTime,
			ClusterCount: 0,
			ItemCount:    len(reports),
		}, nil
	}

	// Prepare cluster items: combine image / clip_text vector (512d) and e5 vector (384d)
	var items []mlclient.ClusterItem
	for _, r := range reports {
		// Attempt to query e5 embedding
		var e5Str *string
		_ = s.pool.QueryRow(ctx, "SELECT embedding::text FROM text_embeddings_e5 WHERE report_type = $1 AND report_id = $2 LIMIT 1", r.repType, r.id).Scan(&e5Str)

		// Attempt to query image embedding or clip_text
		var imgStr *string
		if r.repType == "found" {
			_ = s.pool.QueryRow(ctx, "SELECT embedding::text FROM image_embeddings WHERE report_type = $1 AND report_id = $2 LIMIT 1", r.repType, r.id).Scan(&imgStr)
		}
		if imgStr == nil {
			_ = s.pool.QueryRow(ctx, "SELECT embedding::text FROM text_embeddings_clip WHERE report_type = $1 AND report_id = $2 LIMIT 1", r.repType, r.id).Scan(&imgStr)
		}

		v := parseCombinedVector(imgStr, e5Str)
		items = append(items, mlclient.ClusterItem{
			ID:     fmt.Sprintf("%s:%s", r.repType, r.id),
			Vector: v,
		})
	}

	res, err := s.ml.Cluster(ctx, mlclient.ClusterRequest{
		Items:          items,
		MinClusterSize: 2,
		MinSamples:     1,
	})
	if err != nil {
		finishTime := s.clock.Now()
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO cluster_runs (id, params, versions, status, triggered_by, started_at, finished_at)
			VALUES ($1, $2, $3, 'failed', $4, $5, $6)
		`, runID, paramsBytes, versionsBytes, triggeredBy, startTime, finishTime)
		return nil, fmt.Errorf("ml cluster: %w", err)
	}

	finishTime := s.clock.Now()
	versionsMap := map[string]any{"algorithm": res.Version}
	vBytes, _ := json.Marshal(versionsMap)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin cluster run tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO cluster_runs (id, params, versions, status, triggered_by, started_at, finished_at)
		VALUES ($1, $2, $3, 'done', $4, $5, $6)
	`, runID, paramsBytes, vBytes, triggeredBy, startTime, finishTime)
	if err != nil {
		return nil, fmt.Errorf("insert cluster run: %w", err)
	}

	clusterCount := 0
	clusterSet := make(map[int]bool)
	for _, l := range res.Labels {
		if l.Cluster >= 0 {
			clusterSet[l.Cluster] = true
		}
		parts := strings.SplitN(l.ID, ":", 2)
		if len(parts) != 2 {
			continue
		}
		repType := parts[0]
		repID, err := uuid.Parse(parts[1])
		if err != nil {
			continue
		}

		_, _ = tx.Exec(ctx, `
			INSERT INTO cluster_members (run_id, report_type, report_id, cluster_label, probability)
			VALUES ($1, $2, $3, $4, $5)
		`, runID, repType, repID, l.Cluster, float32(l.Probability))
	}
	clusterCount = len(clusterSet)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cluster run: %w", err)
	}

	return &openapi.ClusterRunSummary{
		Id:           runID,
		Status:       openapi.Done,
		StartedAt:    startTime,
		FinishedAt:   &finishTime,
		ClusterCount: clusterCount,
		ItemCount:    len(items),
	}, nil
}

func parseCombinedVector(imgStr, e5Str *string) []float32 {
	vVisual := parseVector(imgStr, 512)
	vE5 := parseVector(e5Str, 384)

	// Concatenate normalized components (512 + 384 = 896 dimensions)
	out := make([]float32, 896)
	copy(out[:512], vVisual)
	copy(out[512:], vE5)
	return out
}

func parseVector(str *string, dim int) []float32 {
	out := make([]float32, dim)
	if str == nil || *str == "" {
		// Fill small deterministic non-zero seed to avoid all-zero norm
		for i := range out {
			out[i] = 0.001
		}
		return out
	}
	s := strings.Trim(*str, "[] \t\r\n")
	parts := strings.Split(s, ",")
	var norm float64
	for i := 0; i < len(parts) && i < dim; i++ {
		var val float64
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[i]), "%f", &val)
		out[i] = float32(val)
		norm += val * val
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range out {
			out[i] = float32(float64(out[i]) / norm * 0.5)
		}
	}
	return out
}

func (s *Service) GetAiClusters(ctx context.Context, runID *uuid.UUID) (*openapi.AiClustersResponse, error) {
	var targetRunID uuid.UUID
	var status string
	var startedAt time.Time
	var finishedAt *time.Time

	if runID != nil {
		err := s.pool.QueryRow(ctx, "SELECT id, status, started_at, finished_at FROM cluster_runs WHERE id = $1", *runID).Scan(&targetRunID, &status, &startedAt, &finishedAt)
		if err != nil {
			return &openapi.AiClustersResponse{Clusters: []openapi.AiClusterItem{}}, nil
		}
	} else {
		err := s.pool.QueryRow(ctx, "SELECT id, status, started_at, finished_at FROM cluster_runs WHERE status = 'done' ORDER BY started_at DESC LIMIT 1").Scan(&targetRunID, &status, &startedAt, &finishedAt)
		if err != nil {
			return &openapi.AiClustersResponse{Clusters: []openapi.AiClusterItem{}}, nil
		}
	}

	// Fetch members where cluster_label >= 0
	rows, err := s.pool.Query(ctx, `
		SELECT m.cluster_label, m.report_type, m.report_id, m.probability
		FROM cluster_members m
		WHERE m.run_id = $1 AND m.cluster_label >= 0
		ORDER BY m.cluster_label, m.probability DESC
	`, targetRunID)
	if err != nil {
		return nil, fmt.Errorf("query cluster members: %w", err)
	}
	defer rows.Close()

	type rawMem struct {
		label   int
		repType string
		repID   uuid.UUID
		prob    float32
	}
	var rawMembers []rawMem
	for rows.Next() {
		var rm rawMem
		if err := rows.Scan(&rm.label, &rm.repType, &rm.repID, &rm.prob); err == nil {
			rawMembers = append(rawMembers, rm)
		}
	}

	// Group by label
	clusterGroups := make(map[int][]rawMem)
	for _, rm := range rawMembers {
		clusterGroups[rm.label] = append(clusterGroups[rm.label], rm)
	}

	var clusters []openapi.AiClusterItem
	for label, mems := range clusterGroups {
		var members []openapi.AiClusterMember
		var thumbnails []string
		catCounts := make(map[string]int)
		zoneCounts := make(map[string]int)
		var earliest, latest *time.Time

		for _, m := range mems {
			var code, itemName, catName, locName, zoneName string
			var occurredAt time.Time
			var thumbID *uuid.UUID

			if m.repType == "lost" {
				_ = s.pool.QueryRow(ctx, `
					SELECT lr.code, lr.item_name, c.name, COALESCE(l.name, 'Lokasi tidak diketahui'), COALESCE(z.name, 'Kampus'), lr.created_at,
					       (SELECT id FROM media WHERE lost_report_id = lr.id ORDER BY position LIMIT 1)
					FROM lost_reports lr
					JOIN categories c ON c.id = lr.category_id
					LEFT JOIN lost_report_places p ON p.lost_report_id = lr.id AND p.position = 1
					LEFT JOIN locations l ON l.id = p.location_id
					LEFT JOIN zones z ON z.id = l.zone_id
					WHERE lr.id = $1
				`, m.repID).Scan(&code, &itemName, &catName, &locName, &zoneName, &occurredAt, &thumbID)
			} else {
				_ = s.pool.QueryRow(ctx, `
					SELECT fr.code, fr.item_name, c.name, l.name, z.name, fr.found_at,
					       (SELECT id FROM media WHERE found_report_id = fr.id ORDER BY position LIMIT 1)
					FROM found_reports fr
					JOIN categories c ON c.id = fr.category_id
					JOIN locations l ON l.id = fr.found_location_id
					JOIN zones z ON z.id = l.zone_id
					WHERE fr.id = $1
				`, m.repID).Scan(&code, &itemName, &catName, &locName, &zoneName, &occurredAt, &thumbID)
			}

			if code == "" {
				code = m.repID.String()[:8]
			}
			catCounts[catName]++
			zoneCounts[zoneName]++

			if earliest == nil || occurredAt.Before(*earliest) {
				t := occurredAt
				earliest = &t
			}
			if latest == nil || occurredAt.After(*latest) {
				t := occurredAt
				latest = &t
			}

			if thumbID != nil && len(thumbnails) < 4 {
				thumbnails = append(thumbnails, thumbID.String())
			}

			members = append(members, openapi.AiClusterMember{
				ReportType:   openapi.AiClusterMemberReportType(m.repType),
				ReportId:     m.repID,
				Code:         code,
				ItemName:     itemName,
				CategoryName: catName,
				LocationName: locName,
				OccurredAt:   occurredAt,
				Probability:  m.prob,
				ThumbMediaId: thumbID,
			})
		}

		domCat := getDominant(catCounts)
		domZone := getDominant(zoneCounts)

		if thumbnails == nil {
			thumbnails = []string{}
		}
		if members == nil {
			members = []openapi.AiClusterMember{}
		}

		clusters = append(clusters, openapi.AiClusterItem{
			ClusterId:        label,
			Size:             len(members),
			DominantCategory: domCat,
			DominantZone:     domZone,
			EarliestAt:       earliest,
			LatestAt:         latest,
			Thumbnails:       thumbnails,
			Members:          members,
		})
	}

	if clusters == nil {
		clusters = []openapi.AiClusterItem{}
	}

	runSummary := &openapi.ClusterRunSummary{
		Id:           targetRunID,
		Status:       openapi.ClusterRunSummaryStatus(status),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		ClusterCount: len(clusters),
		ItemCount:    len(rawMembers),
	}

	return &openapi.AiClustersResponse{
		Run:      runSummary,
		Clusters: clusters,
	}, nil
}

func getDominant(counts map[string]int) string {
	bestName := ""
	bestCount := -1
	for name, count := range counts {
		if count > bestCount && name != "" {
			bestCount = count
			bestName = name
		}
	}
	if bestName == "" {
		return "Umum"
	}
	return bestName
}

func (s *Service) GetHotspots(ctx context.Context, from, to *time.Time, repType *string, categoryID, zoneID *uuid.UUID) (*openapi.HotspotsResponse, error) {
	now := s.clock.Now()
	periodEnd := now
	if to != nil {
		periodEnd = *to
	}
	periodStart := periodEnd.Add(-30 * 24 * time.Hour)
	if from != nil {
		periodStart = *from
	}

	// Query locations
	locQuery := `
		SELECT l.id, l.name, l.lat, l.lng, z.id, z.name
		FROM locations l
		JOIN zones z ON z.id = l.zone_id
		WHERE l.active = true
	`
	var locArgs []any
	if zoneID != nil {
		locQuery += " AND z.id = $1"
		locArgs = append(locArgs, *zoneID)
	}
	locQuery += " ORDER BY l.name"

	locRows, err := s.pool.Query(ctx, locQuery, locArgs...)
	if err != nil {
		return nil, fmt.Errorf("query hotspot locations: %w", err)
	}
	defer locRows.Close()

	type locData struct {
		id       uuid.UUID
		name     string
		lat, lng float64
		zoneID   uuid.UUID
		zoneName string
	}
	var locations []locData
	for locRows.Next() {
		var l locData
		if err := locRows.Scan(&l.id, &l.name, &l.lat, &l.lng, &l.zoneID, &l.zoneName); err == nil {
			locations = append(locations, l)
		}
	}

	var results []openapi.HotspotLocation
	var insights []string

	for _, l := range locations {
		// Count lost
		var lostCount int64
		var lostArgs = []any{l.id, periodStart, periodEnd}
		lostQ := `
			SELECT COUNT(DISTINCT lr.id)
			FROM lost_reports lr
			JOIN lost_report_places p ON p.lost_report_id = lr.id
			WHERE p.location_id = $1 AND lr.created_at >= $2 AND lr.created_at <= $3
		`
		if categoryID != nil {
			lostQ += " AND lr.category_id = $4"
			lostArgs = append(lostArgs, *categoryID)
		}
		_ = s.pool.QueryRow(ctx, lostQ, lostArgs...).Scan(&lostCount)

		// Count found
		var foundCount int64
		var foundArgs = []any{l.id, periodStart, periodEnd}
		foundQ := `
			SELECT COUNT(fr.id)
			FROM found_reports fr
			WHERE fr.found_location_id = $1 AND fr.found_at >= $2 AND fr.found_at <= $3
		`
		if categoryID != nil {
			foundQ += " AND fr.category_id = $4"
			foundArgs = append(foundArgs, *categoryID)
		}
		_ = s.pool.QueryRow(ctx, foundQ, foundArgs...).Scan(&foundCount)

		// Returned and median time
		var returnedCount int64
		var medianHours *float64
		_ = s.pool.QueryRow(ctx, `
			SELECT COUNT(c.id),
			       PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (c.handed_over_at - fr.found_at)) / 3600.0)
			FROM found_reports fr
			JOIN claims c ON c.found_report_id = fr.id AND c.status = 'handed_over'
			WHERE fr.found_location_id = $1 AND c.handed_over_at >= $2 AND c.handed_over_at <= $3
		`, l.id, periodStart, periodEnd).Scan(&returnedCount, &medianHours)

		total := lostCount + foundCount
		var returnRate float32
		if foundCount > 0 {
			returnRate = float32(math.Round(float64(returnedCount)/float64(foundCount)*100) / 100)
		}

		// BR-PRV-05: Cells under 3 are shown as "< 3"
		formatCount := func(c int64) string {
			if c == 0 {
				return "0"
			}
			if c < 3 {
				return "< 3"
			}
			return fmt.Sprintf("%d", c)
		}

		var medHours32 *float32
		if medianHours != nil {
			v := float32(math.Round(*medianHours*10) / 10)
			medHours32 = &v
		}

		results = append(results, openapi.HotspotLocation{
			LocationId:        l.id,
			LocationName:      l.name,
			ZoneId:            l.zoneID,
			ZoneName:          l.zoneName,
			Lat:               float32(l.lat),
			Lng:               float32(l.lng),
			LostCount:         int(lostCount),
			FoundCount:        int(foundCount),
			TotalCount:        int(total),
			DisplayLost:       formatCount(lostCount),
			DisplayFound:      formatCount(foundCount),
			DisplayTotal:      formatCount(total),
			ReturnRate:        returnRate,
			MedianReturnHours: medHours32,
		})

		// Insight calculation: check if location's count in period exceeds trailing 8 week mean by > 2 std dev and >= 5 reports
		if total >= 5 && len(insights) < 3 {
			var mean8w, stddev8w float64
			err := s.pool.QueryRow(ctx, `
				WITH weeks AS (
					SELECT date_trunc('week', d)::date AS ws
					FROM generate_series($2::date - interval '8 weeks', $2::date - interval '1 day', interval '1 week') d
				),
				weekly_counts AS (
					SELECT w.ws,
					       (SELECT COUNT(*) FROM lost_report_places p JOIN lost_reports lr ON lr.id = p.lost_report_id WHERE p.location_id = $1 AND lr.created_at >= w.ws AND lr.created_at < w.ws + interval '1 week') +
					       (SELECT COUNT(*) FROM found_reports fr WHERE fr.found_location_id = $1 AND fr.found_at >= w.ws AND fr.found_at < w.ws + interval '1 week') AS cnt
					FROM weeks w
				)
				SELECT COALESCE(AVG(cnt), 0), COALESCE(STDDEV(cnt), 0) FROM weekly_counts
			`, l.id, periodStart).Scan(&mean8w, &stddev8w)

			if err == nil && stddev8w > 0 && float64(total) > (mean8w+2*stddev8w) {
				insights = append(insights, fmt.Sprintf("Laporan di %s naik tajam pada periode ini (%d laporan, rata-rata %.1f)", l.name, total, mean8w))
			}
		}
	}

	if insights == nil {
		insights = []string{}
	}
	if results == nil {
		results = []openapi.HotspotLocation{}
	}

	return &openapi.HotspotsResponse{
		Locations: results,
		Insights:  insights,
	}, nil
}

func (s *Service) GetAnalyticsSummary(ctx context.Context) (*openapi.AnalyticsSummary, error) {
	now := s.clock.Now()
	since := now.Add(-30 * 24 * time.Hour)

	var totalLost, totalFound, totalReturned, activeClaims int64
	var periodLost, periodFound, periodReturned int64

	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM lost_reports").Scan(&totalLost)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM found_reports").Scan(&totalFound)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM claims WHERE status = 'handed_over'").Scan(&totalReturned)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM claims WHERE status IN ('submitted', 'under_review', 'info_requested', 'approved')").Scan(&activeClaims)

	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM lost_reports WHERE created_at >= $1", since).Scan(&periodLost)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM found_reports WHERE created_at >= $1", since).Scan(&periodFound)
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM claims WHERE status = 'handed_over' AND handed_over_at >= $1", since).Scan(&periodReturned)

	var returnRate float32
	if totalFound > 0 {
		returnRate = float32(math.Round(float64(totalReturned)/float64(totalFound)*100) / 100)
	}

	return &openapi.AnalyticsSummary{
		TotalLost:      int(totalLost),
		TotalFound:     int(totalFound),
		TotalReturned:  int(totalReturned),
		ReturnRate:     returnRate,
		ActiveClaims:   int(activeClaims),
		PeriodLost:     int(periodLost),
		PeriodFound:    int(periodFound),
		PeriodReturned: int(periodReturned),
	}, nil
}

func (s *Service) GetAnalyticsTrends(ctx context.Context, from, to *time.Time, categoryID, zoneID *uuid.UUID) (*openapi.AnalyticsTrendsResponse, error) {
	now := s.clock.Now()
	targetEnd := now
	if to != nil {
		targetEnd = *to
	}
	targetStart := targetEnd.Add(-84 * 24 * time.Hour) // 12 weeks
	if from != nil {
		targetStart = *from
	}

	// 1. Weekly trend series (12 weeks)
	var weekly []openapi.WeeklyTrend
	for w := 11; w >= 0; w-- {
		wStart := targetEnd.Add(time.Duration(-w*7) * 24 * time.Hour).Truncate(24 * time.Hour)
		wEnd := wStart.Add(7 * 24 * time.Hour)

		var lCount, fCount, rCount int64
		_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM lost_reports WHERE created_at >= $1 AND created_at < $2", wStart, wEnd).Scan(&lCount)
		_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM found_reports WHERE created_at >= $1 AND created_at < $2", wStart, wEnd).Scan(&fCount)
		_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM claims WHERE status = 'handed_over' AND handed_over_at >= $1 AND handed_over_at < $2", wStart, wEnd).Scan(&rCount)

		weekly = append(weekly, openapi.WeeklyTrend{
			WeekStart:     wStart.Format("2006-01-02"),
			LostCount:     int(lCount),
			FoundCount:    int(fCount),
			ReturnedCount: int(rCount),
		})
	}

	// 2. Category breakdown
	catRows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name,
		       (SELECT COUNT(*) FROM lost_reports lr WHERE lr.category_id = c.id AND lr.created_at >= $1 AND lr.created_at <= $2) AS lost_cnt,
		       (SELECT COUNT(*) FROM found_reports fr WHERE fr.category_id = c.id AND fr.created_at >= $1 AND fr.created_at <= $2) AS found_cnt
		FROM categories c
		WHERE c.active = true
		ORDER BY (lost_cnt + found_cnt) DESC
		LIMIT 10
	`, targetStart, targetEnd)
	var categories []openapi.CategoryBreakdown
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var catID uuid.UUID
			var name string
			var lCount, fCount int64
			if err := catRows.Scan(&catID, &name, &lCount, &fCount); err == nil {
				categories = append(categories, openapi.CategoryBreakdown{
					CategoryId:   catID,
					CategoryName: name,
					LostCount:    int(lCount),
					FoundCount:   int(fCount),
				})
			}
		}
	}
	if categories == nil {
		categories = []openapi.CategoryBreakdown{}
	}

	// 3. Hourly weekday grid
	gridRows, err := s.pool.Query(ctx, "SELECT weekday, hour_wib, count FROM mv_hour_weekday ORDER BY weekday, hour_wib")
	var hourly []openapi.HourWeekdayCell
	if err == nil {
		defer gridRows.Close()
		for gridRows.Next() {
			var wd, hr int
			var cnt int64
			if err := gridRows.Scan(&wd, &hr, &cnt); err == nil {
				disp := fmt.Sprintf("%d", cnt)
				if cnt > 0 && cnt < 3 {
					disp = "< 3"
				}
				hourly = append(hourly, openapi.HourWeekdayCell{
					Weekday:      wd,
					Hour:         hr,
					Count:        int(cnt),
					DisplayCount: disp,
				})
			}
		}
	}
	if hourly == nil {
		hourly = []openapi.HourWeekdayCell{}
	}

	// 4. Return rate and median return hours
	var totalFound, totalReturned int64
	var medianHours *float64
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM found_reports WHERE created_at >= $1 AND created_at <= $2", targetStart, targetEnd).Scan(&totalFound)
	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(c.id),
		       PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (c.handed_over_at - fr.found_at)) / 3600.0)
		FROM found_reports fr
		JOIN claims c ON c.found_report_id = fr.id AND c.status = 'handed_over'
		WHERE c.handed_over_at >= $1 AND c.handed_over_at <= $2
	`, targetStart, targetEnd).Scan(&totalReturned, &medianHours)

	var returnRate float32
	if totalFound > 0 {
		returnRate = float32(math.Round(float64(totalReturned)/float64(totalFound)*100) / 100)
	}
	var medHours32 *float32
	if medianHours != nil {
		v := float32(math.Round(*medianHours*10) / 10)
		medHours32 = &v
	}

	return &openapi.AnalyticsTrendsResponse{
		Weekly:            weekly,
		Categories:        categories,
		Hourly:            hourly,
		ReturnRate:        returnRate,
		MedianReturnHours: medHours32,
	}, nil
}

func (s *Service) ExportAnalytics(ctx context.Context, month string) (string, error) {
	locWIB, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		locWIB = time.FixedZone("WIB", 7*3600)
	}

	tStart, err := time.ParseInLocation("2006-01", month, locWIB)
	if err != nil {
		return "", apperr.BadRequest("Format bulan harus YYYY-MM (contoh: 2026-11).")
	}
	tEnd := tStart.AddDate(0, 1, 0)

	type exportRow struct {
		tgl       string
		repType   string
		code      string
		category  string
		location  string
		status    string
		createdAt string
		closedAt  string
	}
	var data []exportRow

	// Lost reports
	lostRows, err := s.pool.Query(ctx, `
		SELECT lr.created_at, lr.code, c.name, COALESCE(l.name, 'Lokasi tidak diketahui'), lr.status, lr.closed_at
		FROM lost_reports lr
		JOIN categories c ON c.id = lr.category_id
		LEFT JOIN lost_report_places p ON p.lost_report_id = lr.id AND p.position = 1
		LEFT JOIN locations l ON l.id = p.location_id
		WHERE lr.created_at >= $1 AND lr.created_at < $2
		ORDER BY lr.created_at
	`, tStart, tEnd)
	if err == nil {
		defer lostRows.Close()
		for lostRows.Next() {
			var catAt time.Time
			var code, catName, locName, stat string
			var closedAt *time.Time
			if err := lostRows.Scan(&catAt, &code, &catName, &locName, &stat, &closedAt); err == nil {
				cStr := ""
				if closedAt != nil {
					cStr = closedAt.In(locWIB).Format(time.RFC3339)
				}
				data = append(data, exportRow{
					tgl:       catAt.In(locWIB).Format("2006-01-02"),
					repType:   "Kehilangan",
					code:      code,
					category:  catName,
					location:  locName,
					status:    stat,
					createdAt: catAt.In(locWIB).Format(time.RFC3339),
					closedAt:  cStr,
				})
			}
		}
	}

	// Found reports
	foundRows, err := s.pool.Query(ctx, `
		SELECT fr.found_at, fr.code, c.name, l.name, fr.custody_status, c2.handed_over_at
		FROM found_reports fr
		JOIN categories c ON c.id = fr.category_id
		JOIN locations l ON l.id = fr.found_location_id
		LEFT JOIN LATERAL (SELECT handed_over_at FROM claims WHERE found_report_id = fr.id AND status = 'handed_over' ORDER BY handed_over_at DESC LIMIT 1) c2 ON true
		WHERE fr.found_at >= $1 AND fr.found_at < $2
		ORDER BY fr.found_at
	`, tStart, tEnd)
	if err == nil {
		defer foundRows.Close()
		for foundRows.Next() {
			var fAt time.Time
			var code, catName, locName, stat string
			var handedAt *time.Time
			if err := foundRows.Scan(&fAt, &code, &catName, &locName, &stat, &handedAt); err == nil {
				hStr := ""
				if handedAt != nil {
					hStr = handedAt.In(locWIB).Format(time.RFC3339)
				}
				data = append(data, exportRow{
					tgl:       fAt.In(locWIB).Format("2006-01-02"),
					repType:   "Ditemukan",
					code:      code,
					category:  catName,
					location:  locName,
					status:    stat,
					createdAt: fAt.In(locWIB).Format(time.RFC3339),
					closedAt:  hStr,
				})
			}
		}
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"Tanggal", "Tipe Laporan", "Nomor Laporan", "Kategori", "Lokasi", "Status", "Waktu Dibuat", "Waktu Selesai"})
	for _, d := range data {
		_ = w.Write([]string{d.tgl, d.repType, d.code, d.category, d.location, d.status, d.createdAt, d.closedAt})
	}
	w.Flush()

	return sb.String(), nil
}
