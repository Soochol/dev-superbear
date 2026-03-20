package marketplace

import (
	"time"

	"github.com/google/uuid"
)

// ItemType enumerates the types of resources that can be shared on the marketplace.
type ItemType string

const (
	ItemTypePipeline        ItemType = "PIPELINE"
	ItemTypeAgentBlock      ItemType = "AGENT_BLOCK"
	ItemTypeSearchPreset    ItemType = "SEARCH_PRESET"
	ItemTypeJudgmentScript  ItemType = "JUDGMENT_SCRIPT"
)

// Status represents the publication status of a marketplace item.
type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusHidden  Status = "HIDDEN"
	StatusRemoved Status = "REMOVED"
)

// UsageAction enumerates trackable user actions.
type UsageAction string

const (
	ActionView    UsageAction = "VIEW"
	ActionFork    UsageAction = "FORK"
	ActionExecute UsageAction = "EXECUTE"
	ActionLike    UsageAction = "LIKE"
)

// SortOption controls the listing sort order.
type SortOption string

const (
	SortRecent      SortOption = "recent"
	SortPopular     SortOption = "popular"
	SortPerformance SortOption = "performance"
	SortMostForked  SortOption = "most_forked"
)

// ---------------------------------------------------------------------------
// Domain Models
// ---------------------------------------------------------------------------

// MarketplaceItem is the core domain entity combining all shared resource types.
type MarketplaceItem struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"userId"`
	Type        ItemType   `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags"`

	// Source resource references (exactly one is non-nil)
	PipelineID       *uuid.UUID `json:"pipelineId,omitempty"`
	AgentBlockID     *uuid.UUID `json:"agentBlockId,omitempty"`
	SearchPresetID   *uuid.UUID `json:"searchPresetId,omitempty"`
	JudgmentScriptID *uuid.UUID `json:"judgmentScriptId,omitempty"`

	// Fork tracking
	ForkedFromID *uuid.UUID `json:"forkedFromId,omitempty"`
	ForkCount    int        `json:"forkCount"`

	// Statistics
	UsageCount int `json:"usageCount"`
	ViewCount  int `json:"viewCount"`
	LikeCount  int `json:"likeCount"`

	// Verification (backtest-based)
	Verified           bool     `json:"verified"`
	BacktestJobID      *uuid.UUID `json:"backtestJobId,omitempty"`
	BacktestWinRate    *float64 `json:"backtestWinRate,omitempty"`
	BacktestAvgReturn  *float64 `json:"backtestAvgReturn,omitempty"`
	BacktestTotalEvents *int    `json:"backtestTotalEvents,omitempty"`

	Status      Status    `json:"status"`
	PublishedAt time.Time `json:"publishedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// MarketplaceLike records a user's like on an item.
type MarketplaceLike struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	ItemID    uuid.UUID `json:"itemId"`
	CreatedAt time.Time `json:"createdAt"`
}

// MarketplaceUsageLog records user interactions.
type MarketplaceUsageLog struct {
	ID        uuid.UUID   `json:"id"`
	UserID    uuid.UUID   `json:"userId"`
	ItemID    uuid.UUID   `json:"itemId"`
	Action    UsageAction `json:"action"`
	CreatedAt time.Time   `json:"createdAt"`
}

// ---------------------------------------------------------------------------
// API DTOs (request / response)
// ---------------------------------------------------------------------------

// ListQuery is the query object for marketplace listing.
type ListQuery struct {
	Type         *ItemType  `form:"type"`
	Sort         SortOption `form:"sort"`
	Search       string     `form:"search"`
	Tags         []string   `form:"tags"`
	VerifiedOnly *bool      `form:"verifiedOnly"`
	Page         int        `form:"page"`
	Limit        int        `form:"limit"`
}

// Defaults fills in zero-value pagination parameters.
func (q *ListQuery) Defaults() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}
	if q.Sort == "" {
		q.Sort = SortRecent
	}
}

// Author is a minimal user representation for display.
type Author struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Avatar *string   `json:"avatar,omitempty"`
}

// ItemStats holds aggregate counters for a marketplace item.
type ItemStats struct {
	UsageCount int `json:"usageCount"`
	ForkCount  int `json:"forkCount"`
	ViewCount  int `json:"viewCount"`
	LikeCount  int `json:"likeCount"`
}

// BacktestStats holds performance metrics attached to a verified item.
type BacktestStats struct {
	WinRate     float64 `json:"winRate"`
	AvgReturn   float64 `json:"avgReturn"`
	TotalEvents int     `json:"totalEvents"`
}

// ForkOrigin provides minimal provenance info for a forked item.
type ForkOrigin struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	AuthorName string    `json:"authorName"`
}

// ItemDetail is the full API response for a single marketplace item.
type ItemDetail struct {
	ID             uuid.UUID      `json:"id"`
	Type           ItemType       `json:"type"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Tags           []string       `json:"tags"`
	Author         Author         `json:"author"`
	Stats          ItemStats      `json:"stats"`
	Verified       bool           `json:"verified"`
	BacktestStats  *BacktestStats `json:"backtestStats,omitempty"`
	ForkedFrom     *ForkOrigin    `json:"forkedFrom,omitempty"`
	IsLikedByMe    bool           `json:"isLikedByMe"`
	PublishedAt    time.Time      `json:"publishedAt"`
}

// ListResponse is the paginated listing response.
type ListResponse struct {
	Items []ItemDetail `json:"items"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
}

// PublishRequest is the request body for publishing a resource to the marketplace.
type PublishRequest struct {
	Type        ItemType `json:"type" binding:"required"`
	ResourceID  string   `json:"resourceId" binding:"required,uuid"`
	Title       string   `json:"title" binding:"required,min=1,max=200"`
	Description string   `json:"description" binding:"required,min=1"`
	Tags        []string `json:"tags"`
}

// ForkResult is the response after a successful fork.
type ForkResult struct {
	NewItemID     uuid.UUID `json:"newItemId"`
	NewResourceID uuid.UUID `json:"newResourceId"`
}

// LikeToggleResult is the response after toggling a like.
type LikeToggleResult struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"likeCount"`
}

// VerifyRequest is the request body for requesting verification.
type VerifyRequest struct {
	BacktestJobID string `json:"backtestJobId" binding:"required,uuid"`
}

// VerificationResult is the response from the verification flow.
type VerificationResult struct {
	Verified bool           `json:"verified"`
	Reason   string         `json:"reason"`
	Stats    *BacktestStats `json:"stats,omitempty"`
}

// VerificationCriteria defines the thresholds for the verified badge.
type VerificationCriteria struct {
	MinTotalEvents int     `json:"minTotalEvents"`
	MinWinRate     float64 `json:"minWinRate"`
	MinAvgReturn   float64 `json:"minAvgReturn"`
}

// DefaultVerificationCriteria returns the production thresholds.
func DefaultVerificationCriteria() VerificationCriteria {
	return VerificationCriteria{
		MinTotalEvents: 10,
		MinWinRate:     50.0,
		MinAvgReturn:   0.0,
	}
}
