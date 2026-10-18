-- name: InsertClaim :exec
INSERT INTO claims (
    id, code, claimant_id, created_by, lost_report_id, found_report_id, match_id, source, statement, answers, preferred_visit,
    status, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(code), sqlc.narg(claimant_id), sqlc.arg(created_by), sqlc.arg(lost_report_id), sqlc.arg(found_report_id),
    sqlc.narg(match_id), sqlc.arg(source), sqlc.arg(statement), sqlc.arg(answers), sqlc.narg(preferred_visit),
    'submitted', sqlc.arg(now), sqlc.arg(now)
);

-- name: ClaimForUpdate :one
SELECT * FROM claims WHERE id = sqlc.narg(id)::uuid OR code = sqlc.narg(code)::text FOR UPDATE;

-- name: GetClaim :one
SELECT * FROM claims WHERE id = sqlc.narg(id)::uuid OR code = sqlc.narg(code)::text;

-- everything a claim page shows, joined once
-- name: ClaimView :one
SELECT c.*,
       fr.code AS found_code, fr.item_name AS found_item_name, fr.description AS found_description, fr.hidden_details,
       fr.colours AS found_colours, fr.brand AS found_brand, fr.found_at, fr.custody_status, fr.status AS found_status,
       fr.holding_post_id, fr.intended_post_id, fr.storage_slot, fr.finder_id, fr.category_id AS found_category_id,
       fc.slug AS found_category_slug, fc.name AS found_category_name, fc.sensitivity AS found_sensitivity,
       coalesce(fp.name, fc.name)::text AS found_category_parent, fc.verification_questions,
       fl.name AS found_location_name,
       lr.code AS lost_code, lr.item_name AS lost_item_name, lr.status AS lost_status, lr.reporter_id AS lost_reporter_id,
       lr.proxy_name, lr.colours AS lost_colours, lr.brand AS lost_brand, lr.private_details, lr.lost_from, lr.lost_until,
       cu.full_name AS claimant_name, cu.email AS claimant_email, cu.phone AS claimant_phone,
       ru.full_name AS reviewer_name, cr.full_name AS creator_name,
       hp.name AS holding_post_name, ip.name AS intended_post_name
FROM claims c
JOIN found_reports fr ON fr.id = c.found_report_id
JOIN categories fc ON fc.id = fr.category_id
LEFT JOIN categories fp ON fp.id = fc.parent_id
JOIN locations fl ON fl.id = fr.found_location_id
JOIN lost_reports lr ON lr.id = c.lost_report_id
LEFT JOIN users cu ON cu.id = c.claimant_id
LEFT JOIN users ru ON ru.id = c.reviewer_id
LEFT JOIN users cr ON cr.id = c.created_by
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN posts ip ON ip.id = fr.intended_post_id
WHERE c.id = sqlc.narg(id)::uuid OR c.code = sqlc.narg(code)::text;

-- name: ListClaims :many
SELECT c.id, c.code, c.status, c.source, c.created_at, c.updated_at, c.approved_expires_at,
       fr.code AS found_code, fr.item_name AS found_item_name, fr.custody_status, fr.holding_post_id,
       lr.code AS lost_code, lr.item_name AS lost_item_name,
       coalesce(cu.full_name, lr.proxy_name, '')::text AS claimant_name,
       hp.name AS holding_post_name,
       coalesce(thumb.id, '00000000-0000-0000-0000-000000000000'::uuid) AS thumb_media_id
FROM claims c
JOIN found_reports fr ON fr.id = c.found_report_id
JOIN lost_reports lr ON lr.id = c.lost_report_id
LEFT JOIN users cu ON cu.id = c.claimant_id
LEFT JOIN posts hp ON hp.id = fr.holding_post_id
LEFT JOIN LATERAL (SELECT m.id FROM media m WHERE m.found_report_id = fr.id AND m.state = 'ready' ORDER BY m.position LIMIT 1) thumb ON true
WHERE (sqlc.narg(claimant_id)::uuid IS NULL OR c.claimant_id = sqlc.narg(claimant_id)::uuid)
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR c.status = ANY(sqlc.arg(statuses)::text[]))
  AND (sqlc.narg(post_id)::uuid IS NULL OR fr.holding_post_id = sqlc.narg(post_id)::uuid OR fr.intended_post_id = sqlc.narg(post_id)::uuid)
  AND (sqlc.narg(found_id)::uuid IS NULL OR c.found_report_id = sqlc.narg(found_id)::uuid)
  AND (sqlc.narg(cursor_created)::timestamptz IS NULL
       OR (c.created_at, c.id) < (sqlc.narg(cursor_created)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountOpenClaimsOf :one
SELECT count(*)::int FROM claims
WHERE claimant_id = $1 AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold');

-- BR-CLM-02: a rejected claim blocks a new one until an officer reopens it
-- name: HasRejectedClaim :one
SELECT EXISTS (SELECT 1 FROM claims WHERE claimant_id = $1 AND found_report_id = $2 AND status = 'rejected')::bool;

-- name: RejectedClaimsSince :one
SELECT count(*)::int FROM claims
WHERE claimant_id = $1 AND status = 'rejected' AND decision_reason IN ('details_mismatch', 'insufficient_evidence', 'suspected_fraud')
  AND decided_at >= $2;

-- name: SetClaimRestricted :exec
UPDATE users SET claim_restricted_at = $2, updated_at = $2 WHERE id = $1 AND claim_restricted_at IS NULL;

-- name: ClearClaimRestriction :exec
UPDATE users SET claim_restricted_at = NULL, updated_at = $2 WHERE id = $1;

-- name: OpenClaimsOfFound :many
SELECT * FROM claims
WHERE found_report_id = $1 AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold')
ORDER BY created_at, id
FOR UPDATE;

-- name: OpenClaimCountOfLost :one
SELECT count(*)::int AS open, count(*) FILTER (WHERE status = 'approved')::int AS approved FROM claims
WHERE lost_report_id = $1 AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold');

-- name: OpenClaimCountOfFound :one
SELECT count(*)::int AS open, count(*) FILTER (WHERE status = 'approved')::int AS approved FROM claims
WHERE found_report_id = $1 AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold');

-- name: PickUpClaim :exec
UPDATE claims SET status = 'under_review', reviewer_id = $2, updated_at = $3, version = version + 1 WHERE id = $1;

-- name: SetClaimInfoRequested :exec
UPDATE claims SET status = 'info_requested', info_requested_at = $2, updated_at = $2, version = version + 1 WHERE id = $1;

-- name: SetClaimUnderReview :exec
UPDATE claims SET status = 'under_review', info_requested_at = NULL, updated_at = $2, version = version + 1 WHERE id = $1;

-- name: SetClaimOnHold :exec
UPDATE claims SET status = 'on_hold', updated_at = $2, version = version + 1 WHERE id = $1;

-- name: ApproveClaim :exec
UPDATE claims
SET status = 'approved', decided_by = $2, decided_at = $3, decision_note = sqlc.narg(decision_note), decision_reason = NULL,
    handover_nonce = $4, handover_code_issued_at = $3, handover_failed_attempts = 0, handover_locked_at = NULL,
    approved_expires_at = $5, expiring_notified_at = NULL, updated_at = $3, version = version + 1
WHERE id = $1;

-- name: RejectClaim :exec
UPDATE claims
SET status = 'rejected', decision_reason = $2, decision_note = sqlc.narg(decision_note), decided_by = sqlc.narg(decided_by), decided_at = $3,
    handover_nonce = NULL, approved_expires_at = NULL, updated_at = $3, version = version + 1
WHERE id = $1;

-- name: EndClaim :exec
UPDATE claims
SET status = $2, decision_note = coalesce(sqlc.narg(decision_note), decision_note), handover_nonce = NULL, approved_expires_at = NULL,
    expiring_notified_at = NULL, updated_at = $3, version = version + 1
WHERE id = $1;

-- K13 and K15 return a claim to review
-- name: ReopenClaim :exec
UPDATE claims
SET status = 'under_review', decision_reason = NULL, handover_nonce = NULL, approved_expires_at = NULL, expiring_notified_at = NULL,
    reopened_at = $2, updated_at = $2, version = version + 1
WHERE id = $1;

-- name: HandOverClaim :exec
UPDATE claims
SET status = 'handed_over', handed_over_at = $2, handed_over_by = $3, recipient_id_type = $4, recipient_id_last4 = $5,
    recipient_name_confirmed = $6, handover_nonce = NULL, approved_expires_at = NULL, updated_at = $2, version = version + 1
WHERE id = $1;

-- name: ClaimCodeFailed :one
UPDATE claims
SET handover_failed_attempts = handover_failed_attempts + 1,
    handover_locked_at = CASE WHEN handover_failed_attempts + 1 >= sqlc.arg(max_attempts)::int THEN sqlc.arg(now)::timestamptz ELSE handover_locked_at END,
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING handover_failed_attempts, handover_locked_at;

-- name: RegenerateHandoverCode :exec
UPDATE claims
SET handover_nonce = $2, handover_code_issued_at = $3, handover_failed_attempts = 0, handover_locked_at = NULL,
    approved_expires_at = $4, expiring_notified_at = NULL, updated_at = $3, version = version + 1
WHERE id = $1;

-- name: SetClaimInternalNote :exec
UPDATE claims SET internal_note = $2, updated_at = $3 WHERE id = $1;

-- name: InsertClaimMessage :exec
INSERT INTO claim_messages (id, claim_id, author_id, author_role, body, created_at) VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListClaimMessages :many
SELECT m.id, m.author_id, m.author_role, m.body, m.created_at, coalesce(u.full_name, 'Sistem')::text AS author_name
FROM claim_messages m LEFT JOIN users u ON u.id = m.author_id
WHERE m.claim_id = $1 ORDER BY m.created_at, m.id;

-- name: InsertClaimCheck :exec
INSERT INTO claim_checks (claim_id, item_key, position, label, required, info_only, suggestion)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (claim_id, item_key) DO UPDATE
SET label = excluded.label, required = excluded.required, info_only = excluded.info_only, suggestion = excluded.suggestion;

-- name: ListClaimChecks :many
SELECT * FROM claim_checks WHERE claim_id = $1 ORDER BY position;

-- name: SetClaimCheck :execrows
UPDATE claim_checks SET result = $3, note = sqlc.narg(note), decided_by = $4, decided_at = $5
WHERE claim_id = $1 AND item_key = $2;

-- name: DeleteStaleClaimChecks :exec
DELETE FROM claim_checks WHERE claim_id = $1 AND NOT (item_key = ANY(sqlc.arg(keep)::text[]));

-- the pair's scores for the checklist, null when the pair was never stored
-- name: PairScores :one
SELECT id, s_semantic, s_attr, s_time, s_location, overall, tier, flags FROM matches WHERE lost_report_id = $1 AND found_report_id = $2;

-- name: SetPairStatus :exec
UPDATE matches SET status = $3, version = version + 1
WHERE lost_report_id = $1 AND found_report_id = $2 AND status = ANY(sqlc.arg(from_statuses)::text[]);

-- name: FoundMediaAttributes :many
SELECT ia.colours, ia.brands, ia.category_scores FROM image_attributes ia JOIN media m ON m.id = ia.media_id
WHERE m.found_report_id = $1 AND m.state = 'ready';

-- name: OtherOpenClaimsOfFound :many
SELECT id, code FROM claims
WHERE found_report_id = $1 AND id <> $2 AND status IN ('submitted', 'under_review', 'info_requested', 'approved', 'on_hold');

-- name: ClaimsDueToExpire :many
SELECT id FROM claims
WHERE (status = 'info_requested' AND info_requested_at <= sqlc.arg(info_before)::timestamptz)
   OR (status = 'approved' AND approved_expires_at <= sqlc.arg(now)::timestamptz)
ORDER BY updated_at
LIMIT 200;

-- name: ClaimsExpiringSoon :many
SELECT c.id FROM claims c
WHERE c.status = 'approved' AND c.approved_expires_at <= sqlc.arg(until)::timestamptz AND c.approved_expires_at > sqlc.arg(now)::timestamptz
  AND c.expiring_notified_at IS NULL
ORDER BY c.approved_expires_at
LIMIT 200;

-- name: MarkClaimExpiringNotified :exec
UPDATE claims SET expiring_notified_at = $2 WHERE id = $1;

-- Reports after a claim moves

-- name: ForceLostStatus :exec
UPDATE lost_reports SET status = $2, last_activity_at = $3 WHERE id = $1;

-- name: ForceFoundStatus :exec
UPDATE found_reports SET status = $2 WHERE id = $1;

-- name: LockLostStatus :one
SELECT status FROM lost_reports WHERE id = $1 FOR UPDATE;

-- name: LockFoundStatus :one
SELECT status, custody_status FROM found_reports WHERE id = $1 FOR UPDATE;

-- name: ReleaseFoundReport :exec
UPDATE found_reports SET status = 'returned', custody_status = 'released', updated_at = $2, version = version + 1 WHERE id = $1;

-- name: ReturnLostReport :exec
UPDATE lost_reports SET status = 'returned', last_activity_at = $2, updated_at = $2, version = version + 1 WHERE id = $1;

-- name: CandidatesOfLost :one
SELECT count(*)::int FROM matches
WHERE lost_report_id = $1 AND status = 'active' AND tier IN ('high', 'medium') AND rank_for_lost IS NOT NULL AND owner_dismissed_at IS NULL;

-- name: CandidatesOfFound :one
SELECT count(*)::int FROM matches
WHERE found_report_id = $1 AND status = 'active' AND tier IN ('high', 'medium') AND rank_for_found IS NOT NULL;

-- name: UserBrief :one
SELECT id, full_name, email, phone, nim_nip FROM users WHERE id = $1;

-- history of the claimant for V8
-- name: ClaimantRejectedCount90 :one
SELECT count(*)::int FROM claims WHERE claimant_id = $1 AND status = 'rejected' AND decided_at >= $2;
