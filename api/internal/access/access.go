package access

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/pisondev/simpanugm/api/internal/platform/apperr"
)

type Role string

const (
	Member  Role = "member"
	Officer Role = "officer"
	Admin   Role = "admin"
)

func (r Role) Valid() bool { return r == Member || r == Officer || r == Admin }

type Permission string

const (
	AccountSelfRead      Permission = "account.self.read"
	AccountSelfUpdate    Permission = "account.self.update"
	AccountSessions      Permission = "account.sessions.manage"
	ReportLostCreate     Permission = "report.lost.create"
	ReportLostRead       Permission = "report.lost.read"
	ReportLostUpdate     Permission = "report.lost.update"
	ReportLostClose      Permission = "report.lost.close"
	ReportFoundCreate    Permission = "report.found.create"
	ReportFoundRead      Permission = "report.found.read"
	ReportFoundUpdate    Permission = "report.found.update"
	ReportFoundClose     Permission = "report.found.close"
	ReportPrivateRead    Permission = "report.private.read"
	ReportHistoryRead    Permission = "report.history.read"
	ReportProxyCreate    Permission = "report.proxy.create"
	MediaUpload          Permission = "media.upload"
	MediaRead            Permission = "media.read"
	MediaDelete          Permission = "media.delete"
	MasterdataRead       Permission = "masterdata.read"
	MasterdataManage     Permission = "masterdata.manage"
	MatchRead            Permission = "match.read"
	MatchFeedback        Permission = "match.feedback"
	MatchDismiss         Permission = "match.dismiss"
	MatchRematch         Permission = "match.rematch"
	AssistAnalyze        Permission = "assist.analyze"
	CustodyReceive       Permission = "custody.receive"
	CustodyInventoryRead Permission = "custody.inventory.read"
	SearchReports        Permission = "search.reports"
	OfficerQueueRead     Permission = "officer.queue.read"
	UserRead             Permission = "user.read"
	UserInvite           Permission = "user.invite"
	UserUpdateRole       Permission = "user.update_role"
	UserAssignPost       Permission = "user.assign_post"
	ClaimCreate          Permission = "claim.create"
	ClaimRead            Permission = "claim.read"
	ClaimWithdraw        Permission = "claim.withdraw"
	ClaimMessage         Permission = "claim.message"
	ClaimReview          Permission = "claim.review"
	ClaimApprove         Permission = "claim.approve"
	ClaimInternalNote    Permission = "claim.internal_note"
	CustodyHandover      Permission = "custody.handover"
	CustodyTransfer      Permission = "custody.transfer"
	CustodyExtend        Permission = "custody.extend"
	CustodyDispose       Permission = "custody.dispose"
)

var memberPerms = []Permission{
	AccountSelfRead, AccountSelfUpdate, AccountSessions,
	ReportLostCreate, ReportLostRead, ReportLostUpdate, ReportLostClose,
	ReportFoundCreate, ReportFoundRead, ReportFoundUpdate, ReportFoundClose,
	ReportPrivateRead, ReportHistoryRead,
	MediaUpload, MediaRead, MediaDelete,
	MasterdataRead,
	MatchRead, MatchFeedback, MatchDismiss, AssistAnalyze,
	ClaimCreate, ClaimRead, ClaimWithdraw, ClaimMessage,
}

var officerPerms = append(slices.Clone(memberPerms),
	ReportProxyCreate, CustodyReceive, CustodyInventoryRead, SearchReports, OfficerQueueRead, MatchRematch,
	ClaimReview, ClaimApprove, ClaimInternalNote, CustodyHandover, CustodyTransfer, CustodyExtend, CustodyDispose,
)

var adminPerms = append(slices.Clone(officerPerms),
	UserRead, UserInvite, UserUpdateRole, UserAssignPost, MasterdataManage,
)

var rolePerms = map[Role][]Permission{Member: memberPerms, Officer: officerPerms, Admin: adminPerms}

func (r Role) Can(p Permission) bool { return slices.Contains(rolePerms[r], p) }

func (r Role) Permissions() []string {
	out := make([]string, 0, len(rolePerms[r]))
	for _, p := range rolePerms[r] {
		out = append(out, string(p))
	}
	slices.Sort(out)
	return out
}

type Principal struct {
	UserID          uuid.UUID
	SessionID       uuid.UUID
	Role            Role
	Status          string
	Email           string
	FullName        string
	EmailVerified   bool
	ClaimRestricted bool
	PostIDs         []uuid.UUID
}

func (p *Principal) IsStaff() bool { return p != nil && (p.Role == Officer || p.Role == Admin) }

func (p *Principal) IsAdmin() bool { return p != nil && p.Role == Admin }

func (p *Principal) AssignedTo(post uuid.UUID) bool {
	return p != nil && (p.Role == Admin || slices.Contains(p.PostIDs, post))
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func FromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

type Rule struct {
	Public     bool
	Permission Permission
	Verified   bool
}

// one rule per operationId in api/openapi.yaml, a test keeps the two in step
var Operations = map[string]Rule{
	"register":                   {Public: true},
	"verifyEmail":                {Public: true},
	"resendVerification":         {Permission: AccountSelfRead},
	"login":                      {Public: true},
	"logout":                     {Permission: AccountSelfRead},
	"forgotPassword":             {Public: true},
	"resetPassword":              {Public: true},
	"acceptInvitation":           {Public: true},
	"getMe":                      {Permission: AccountSelfRead},
	"listCategories":             {Permission: MasterdataRead},
	"listZones":                  {Permission: MasterdataRead},
	"listLocations":              {Permission: MasterdataRead},
	"listPosts":                  {Public: true},
	"listColours":                {Permission: MasterdataRead},
	"uploadMedia":                {Permission: MediaUpload, Verified: true},
	"getMedia":                   {Permission: MediaRead},
	"deleteMedia":                {Permission: MediaDelete},
	"createLostReport":           {Permission: ReportLostCreate, Verified: true},
	"listLostReports":            {Permission: ReportLostRead},
	"getLostReport":              {Permission: ReportLostRead},
	"updateLostReport":           {Permission: ReportLostUpdate, Verified: true},
	"closeLostReport":            {Permission: ReportLostClose},
	"getLostReportHistory":       {Permission: ReportHistoryRead},
	"createFoundReport":          {Permission: ReportFoundCreate, Verified: true},
	"listFoundReports":           {Permission: ReportFoundRead},
	"listPublicFoundReports":     {Permission: ReportFoundRead},
	"getFoundReport":             {Permission: ReportFoundRead},
	"updateFoundReport":          {Permission: ReportFoundUpdate, Verified: true},
	"closeFoundReport":           {Permission: ReportFoundClose},
	"getFoundReportHistory":      {Permission: ReportHistoryRead},
	"receiveFoundReport":         {Permission: CustodyReceive},
	"getPostInventory":           {Permission: CustodyInventoryRead},
	"searchReports":              {Permission: SearchReports},
	"adminListUsers":             {Permission: UserRead},
	"adminInviteUser":            {Permission: UserInvite},
	"adminUpdateUser":            {Permission: UserUpdateRole},
	"listNotifications":          {Permission: AccountSelfRead},
	"countUnreadNotifications":   {Permission: AccountSelfRead},
	"markAllNotificationsRead":   {Permission: AccountSelfRead},
	"markNotificationRead":       {Permission: AccountSelfRead},
	"streamNotifications":        {Permission: AccountSelfRead},
	"getNotificationPreferences": {Permission: AccountSelfRead},
	"putNotificationPreferences": {Permission: AccountSelfUpdate},
	"updateMe":                   {Permission: AccountSelfUpdate},
	"startEmailChange":           {Permission: AccountSelfUpdate},
	"confirmEmailChange":         {Public: true},
	"changePassword":             {Permission: AccountSelfUpdate},
	"listMySessions":             {Permission: AccountSessions},
	"revokeMySession":            {Permission: AccountSessions},
	"listLostReportMatches":      {Permission: MatchRead},
	"listFoundReportMatches":     {Permission: OfficerQueueRead},
	"getFoundReportMatchHint":    {Permission: ReportFoundRead},
	"getMatch":                   {Permission: MatchRead},
	"submitMatchFeedback":        {Permission: MatchFeedback},
	"dismissMatch":               {Permission: MatchDismiss},
	"rematchReport":              {Permission: MatchRematch},
	"assistAnalyzeImage":         {Permission: AssistAnalyze},
	"assistDescribe":             {Permission: AssistAnalyze},
	"createClaim":                {Permission: ClaimCreate, Verified: true},
	"listClaims":                 {Permission: ClaimRead},
	"getClaim":                   {Permission: ClaimRead},
	"pickUpClaim":                {Permission: ClaimReview},
	"requestClaimInfo":           {Permission: ClaimReview},
	"respondToClaim":             {Permission: ClaimWithdraw},
	"getClaimChecks":             {Permission: ClaimReview},
	"putClaimChecks":             {Permission: ClaimReview},
	"approveClaim":               {Permission: ClaimApprove},
	"rejectClaim":                {Permission: ClaimReview},
	"revokeClaim":                {Permission: ClaimApprove},
	"reopenClaim":                {Permission: ClaimReview},
	"withdrawClaim":              {Permission: ClaimWithdraw},
	"regenerateHandoverCode":     {Permission: CustodyHandover},
	"listClaimMessages":          {Permission: ClaimMessage},
	"postClaimMessage":           {Permission: ClaimMessage},
	"setClaimInternalNote":       {Permission: ClaimInternalNote},
	"handoverFoundReport":        {Permission: CustodyHandover},
	"adminListCategories":        {Permission: MasterdataManage},
	"adminCreateCategory":        {Permission: MasterdataManage},
	"adminUpdateCategory":        {Permission: MasterdataManage},
	"adminListCompatibility":     {Permission: MasterdataManage},
	"adminPutCompatibility":      {Permission: MasterdataManage},
	"adminListZones":             {Permission: MasterdataManage},
	"adminCreateZone":            {Permission: MasterdataManage},
	"adminUpdateZone":            {Permission: MasterdataManage},
	"adminListLocations":         {Permission: MasterdataManage},
	"adminCreateLocation":        {Permission: MasterdataManage},
	"adminUpdateLocation":        {Permission: MasterdataManage},
	"adminListPosts":             {Permission: MasterdataManage},
	"adminCreatePost":            {Permission: MasterdataManage},
	"adminUpdatePost":            {Permission: MasterdataManage},
}

func Check(p *Principal, operationID string) error {
	rule, ok := Operations[operationID]
	if !ok {
		return apperr.Forbidden()
	}
	if rule.Public {
		return nil
	}
	if p == nil {
		return apperr.Unauthenticated()
	}
	if !p.Role.Can(rule.Permission) {
		return apperr.Forbidden()
	}
	if rule.Verified && !p.EmailVerified {
		return apperr.AccountUnverified()
	}
	return nil
}
