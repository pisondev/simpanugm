package demo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pisondev/simpanugm/api/internal/access"
	"github.com/pisondev/simpanugm/api/internal/claims"
	"github.com/pisondev/simpanugm/api/internal/openapi"
	"github.com/pisondev/simpanugm/api/internal/reports"
)

func seedClaimState(ctx context.Context, svc *claims.Service, rep *reports.Service, officer, owner *access.Principal, seed DemoClaimSeed, claim openapi.Claim, foundCode, lostCode string) error {
	if seed.TargetStatus == "submitted" {
		return nil
	}
	if _, err := svc.PickUp(ctx, officer, claim.Code); err != nil {
		return fmt.Errorf("pick up: %w", err)
	}
	switch seed.TargetStatus {
	case "under_review":
		return nil
	case "info_requested":
		_, err := svc.RequestInfo(ctx, officer, claim.Code, seed.RequestMessage)
		return err
	case "rejected":
		_, err := svc.Reject(ctx, officer, claim.Code, seed.DecisionReason, &seed.DecisionNote)
		return err
	case "approved", "completed":
		list, err := svc.Checks(ctx, officer, claim.Code)
		if err != nil {
			return err
		}
		var checks []claims.CheckResult
		for _, it := range list.Items {
			if !it.InfoOnly {
				checks = append(checks, claims.CheckResult{ItemKey: it.ItemKey, Result: "pass"})
			}
		}
		if _, err := svc.PutChecks(ctx, officer, claim.Code, checks); err != nil {
			return err
		}
		current, err := svc.Get(ctx, officer, claim.Code)
		if err != nil {
			return err
		}
		approved, err := svc.Approve(ctx, officer, claim.Code, strconv.Itoa(current.Version), &seed.DecisionNote)
		if err != nil {
			return fmt.Errorf("approve: %w", err)
		}
		if seed.TargetStatus == "approved" {
			return nil
		}
		view, err := svc.Get(ctx, owner, approved.Code)
		if err != nil {
			return err
		}
		if view.HandoverCode == nil {
			return fmt.Errorf("approved claim %s has no owner handover code", claim.Code)
		}
		if _, err := svc.Handover(ctx, officer, foundCode, claims.HandoverInput{
			ClaimID: approved.Id, Code: *view.HandoverCode, IDType: "ktm", IDLast4: "3107", NameConfirmed: true,
		}); err != nil {
			return err
		}
		_, err = rep.ConfirmReceipt(ctx, owner, lostCode)
		return err
	default:
		return fmt.Errorf("unknown demo claim target %q", seed.TargetStatus)
	}
}
