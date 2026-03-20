# 알림 시스템 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 모니터링 이벤트 발생 시 In-App(SSE), Web Push, Slack/Telegram 채널로 사용자에게 실시간 알림을 전달하는 멀티채널 알림 시스템을 구축한다.
**Architecture:** 모니터링 엔진(Plan 6)에서 발생한 이벤트를 중앙 NotificationService가 수신하여 Notification 레코드를 생성한 뒤, 채널별 디스패처(SSE, Web Push, Webhook)로 분배한다. SSE는 Go Gin 핸들러의 `/api/notifications/stream` 엔드포인트로 실시간 푸시하고, Web Push는 VAPID 키 기반 브라우저 알림, Slack/Telegram은 webhook/bot API를 호출한다.
**Tech Stack:** Go (Gin), sqlc (PostgreSQL), asynq (Redis job queue), Server-Sent Events (SSE), Go web push (VAPID), Slack Webhook, Telegram Bot API

**Frontend (유지):** `src/features/notification/` — 알림 UI 컴포넌트(알림 목록, 설정 패널)는 기존 Next.js 프론트엔드에 유지하되, API 호출 경로를 Go 백엔드로 변경한다.

**Go Backend Layer:**
```
backend/internal/
  handler/notification_handler.go   # GET /notifications, POST /read, GET /stream (SSE), settings
  service/notification_service.go   # 알림 생성 + 채널 디스패치 (emitter + dispatcher)
  service/notification_rules.go     # 규칙 엔진 (필터링)
  repository/notification_repo.go   # Notification sqlc CRUD
  infra/sse/manager.go              # SSE 연결 관리
  infra/telegram/bot.go             # Telegram Bot API
  infra/slack/webhook.go            # Slack Incoming Webhook
  infra/webpush/sender.go           # Web Push (VAPID)
```

**Deployment Note:** SSE requires long-lived connections. Go Gin natively supports SSE with `c.Stream()`. This is compatible with Cloud Run (with timeout config) or dedicated servers.

---

## 의존성

- **Plan 6 (모니터링 엔진)**: 이벤트 발생 시 `NotificationService.Emit()` 호출 지점

---

## Task 1: Notification 데이터 모델 및 기본 CRUD

알림 저장, 조회, 읽음 처리를 위한 sqlc 쿼리와 Go repository, 기본 Gin 핸들러를 구현한다.

**Files:**
- Create: `backend/internal/repository/sqlc/migrations/007_add_notification_models.sql`
- Create: `backend/internal/repository/sqlc/queries/notification.sql`
- Create: `backend/internal/repository/notification_repo.go`
- Create: `backend/internal/handler/notification_handler.go`
- Create: `backend/internal/handler/notification_handler_test.go`

**Steps:**

- [ ] PostgreSQL 마이그레이션: Notification, NotificationPreference, PushSubscription 테이블 추가

```sql
-- backend/internal/repository/sqlc/migrations/007_add_notification_models.sql

-- 알림 타입 enum
CREATE TYPE notification_type AS ENUM (
  'PRICE_ALERT',
  'SUCCESS_CONDITION',
  'FAILURE_CONDITION',
  'NEWS_EVENT',
  'DISCLOSURE_EVENT',
  'SECTOR_EVENT',
  'MONITOR_RESULT'
);

-- 알림 테이블
CREATE TABLE notifications (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id),
  case_id     UUID REFERENCES cases(id),
  type        notification_type NOT NULL,
  title       TEXT NOT NULL,
  body        TEXT NOT NULL,
  data        JSONB,
  read        BOOLEAN NOT NULL DEFAULT false,
  read_at     TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_read_created
  ON notifications(user_id, read, created_at DESC);

-- 알림 설정 테이블
CREATE TABLE notification_preferences (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID NOT NULL UNIQUE REFERENCES users(id),
  in_app_enabled    BOOLEAN NOT NULL DEFAULT true,
  push_enabled      BOOLEAN NOT NULL DEFAULT false,
  slack_enabled     BOOLEAN NOT NULL DEFAULT false,
  telegram_enabled  BOOLEAN NOT NULL DEFAULT false,
  slack_webhook_url TEXT,
  telegram_chat_id  TEXT,
  min_impact_score  INT NOT NULL DEFAULT 5,
  muted_case_ids    UUID[] NOT NULL DEFAULT '{}',
  muted_types       notification_type[] NOT NULL DEFAULT '{}',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Web Push 구독 테이블
CREATE TABLE push_subscriptions (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id),
  endpoint   TEXT NOT NULL,
  p256dh     TEXT NOT NULL,
  auth       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subscriptions_user ON push_subscriptions(user_id);
```

- [ ] sqlc 쿼리 정의 — Notification CRUD

```sql
-- backend/internal/repository/sqlc/queries/notification.sql

-- name: CreateNotification :one
INSERT INTO notifications (user_id, case_id, type, title, body, data)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
  AND (sqlc.narg('unread_only')::boolean IS NULL OR read = false)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1
  AND (sqlc.narg('unread_only')::boolean IS NULL OR read = false);

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1 AND read = false;

-- name: MarkAsRead :one
UPDATE notifications
SET read = true, read_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkAllAsRead :exec
UPDATE notifications
SET read = true, read_at = now()
WHERE user_id = $1 AND read = false;

-- name: GetPreferences :one
SELECT * FROM notification_preferences
WHERE user_id = $1;

-- name: UpsertPreferences :one
INSERT INTO notification_preferences (
  user_id, in_app_enabled, push_enabled, slack_enabled, telegram_enabled,
  slack_webhook_url, telegram_chat_id, min_impact_score, muted_case_ids, muted_types
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id) DO UPDATE SET
  in_app_enabled = EXCLUDED.in_app_enabled,
  push_enabled = EXCLUDED.push_enabled,
  slack_enabled = EXCLUDED.slack_enabled,
  telegram_enabled = EXCLUDED.telegram_enabled,
  slack_webhook_url = EXCLUDED.slack_webhook_url,
  telegram_chat_id = EXCLUDED.telegram_chat_id,
  min_impact_score = EXCLUDED.min_impact_score,
  muted_case_ids = EXCLUDED.muted_case_ids,
  muted_types = EXCLUDED.muted_types,
  updated_at = now()
RETURNING *;

-- name: ListPushSubscriptions :many
SELECT * FROM push_subscriptions WHERE user_id = $1;

-- name: CreatePushSubscription :one
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE id = $1;

-- name: DeletePushSubscriptionsByIDs :exec
DELETE FROM push_subscriptions WHERE id = ANY($1::uuid[]);
```

- [ ] Go Repository — sqlc 생성 코드를 래핑

```go
// backend/internal/repository/notification_repo.go
package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	db "nexus/backend/internal/repository/sqlc"
)

type NotificationRepo struct {
	q *db.Queries
}

func NewNotificationRepo(q *db.Queries) *NotificationRepo {
	return &NotificationRepo{q: q}
}

type CreateNotificationParams struct {
	UserID string
	CaseID *string
	Type   string
	Title  string
	Body   string
	Data   map[string]interface{}
}

func (r *NotificationRepo) Create(ctx context.Context, p CreateNotificationParams) (db.Notification, error) {
	var dataJSON json.RawMessage
	if p.Data != nil {
		b, err := json.Marshal(p.Data)
		if err != nil {
			return db.Notification{}, err
		}
		dataJSON = b
	}

	var caseID uuid.NullUUID
	if p.CaseID != nil {
		uid, err := uuid.Parse(*p.CaseID)
		if err != nil {
			return db.Notification{}, err
		}
		caseID = uuid.NullUUID{UUID: uid, Valid: true}
	}

	return r.q.CreateNotification(ctx, db.CreateNotificationParams{
		UserID: uuid.MustParse(p.UserID),
		CaseID: caseID,
		Type:   db.NotificationType(p.Type),
		Title:  p.Title,
		Body:   p.Body,
		Data:   dataJSON,
	})
}

type ListParams struct {
	UserID     string
	UnreadOnly bool
	Page       int
	Limit      int
}

type ListResult struct {
	Notifications []db.Notification
	UnreadCount   int64
	Total         int64
}

func (r *NotificationRepo) List(ctx context.Context, p ListParams) (ListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	offset := (p.Page - 1) * p.Limit

	userID := uuid.MustParse(p.UserID)

	var unreadOnly *bool
	if p.UnreadOnly {
		unreadOnly = &p.UnreadOnly
	}

	notifications, err := r.q.ListNotifications(ctx, db.ListNotificationsParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		Limit:      int32(p.Limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return ListResult{}, err
	}

	unreadCount, err := r.q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}

	total, err := r.q.CountNotifications(ctx, db.CountNotificationsParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
	})
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Notifications: notifications,
		UnreadCount:   unreadCount,
		Total:         total,
	}, nil
}

func (r *NotificationRepo) MarkAsRead(ctx context.Context, id string) (db.Notification, error) {
	return r.q.MarkAsRead(ctx, uuid.MustParse(id))
}

func (r *NotificationRepo) MarkAllAsRead(ctx context.Context, userID string) error {
	return r.q.MarkAllAsRead(ctx, uuid.MustParse(userID))
}

func (r *NotificationRepo) GetPreferences(ctx context.Context, userID string) (db.NotificationPreference, error) {
	return r.q.GetPreferences(ctx, uuid.MustParse(userID))
}

func (r *NotificationRepo) CountUnread(ctx context.Context, userID string) (int64, error) {
	return r.q.CountUnreadNotifications(ctx, uuid.MustParse(userID))
}
```

- [ ] Gin 핸들러 — 알림 목록 조회, 읽음 처리 (thin controller)

```go
// backend/internal/handler/notification_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nexus/backend/internal/repository"
	"nexus/backend/internal/service"
	"nexus/backend/internal/infra/sse"
)

type NotificationHandler struct {
	repo       *repository.NotificationRepo
	service    *service.NotificationService
	sseManager *sse.Manager
}

func NewNotificationHandler(
	repo *repository.NotificationRepo,
	svc *service.NotificationService,
	sseMgr *sse.Manager,
) *NotificationHandler {
	return &NotificationHandler{
		repo:       repo,
		service:    svc,
		sseManager: sseMgr,
	}
}

// GET /api/notifications?page=1&limit=20&unreadOnly=true
func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unreadOnly") == "true"

	result, err := h.repo.List(c.Request.Context(), repository.ListParams{
		UserID:     userID,
		UnreadOnly: unreadOnly,
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": result.Notifications,
		"unreadCount":   result.UnreadCount,
		"total":         result.Total,
	})
}

// PUT /api/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")

	_, err := h.repo.MarkAsRead(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PUT /api/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("userId")

	err := h.repo.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GET /api/notifications/unread-count
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := c.GetString("userId")

	count, err := h.repo.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}
```

- [ ] 라우트 등록 (router.go에 추가)

```go
// backend/internal/handler/router.go (기존 파일에 추가)
func RegisterNotificationRoutes(rg *gin.RouterGroup, h *NotificationHandler) {
	notifications := rg.Group("/notifications")
	{
		notifications.GET("", h.List)
		notifications.PUT("/:id/read", h.MarkAsRead)
		notifications.PUT("/read-all", h.MarkAllAsRead)
		notifications.GET("/unread-count", h.UnreadCount)
		notifications.GET("/stream", h.Stream)          // Task 3
		notifications.GET("/settings", h.GetSettings)    // Task 5
		notifications.PUT("/settings", h.UpdateSettings)  // Task 5
		notifications.POST("/push/subscribe", h.PushSubscribe)  // Task 4
		notifications.GET("/push/vapid-key", h.VAPIDKey)        // Task 4
	}
}
```

- [ ] 테스트: 알림 생성 -> 목록 조회 -> 읽음 처리 -> unreadCount 감소 확인

```bash
cd backend && go test ./internal/handler/ -run TestNotificationCRUD -v
git add backend/internal/repository/ backend/internal/handler/
git commit -m "feat(alerts): Notification 데이터 모델 및 기본 CRUD API (Go/sqlc)"
```

---

## Task 2: NotificationService 및 채널 디스패처

모니터링 이벤트를 수신하여 사용자 설정에 따라 채널별로 분배하는 중앙 서비스를 구현한다. TypeScript의 emitter.ts + dispatcher.ts를 단일 Go 서비스로 통합한다.

**Files:**
- Create: `backend/internal/service/notification_service.go`
- Create: `backend/internal/service/notification_service_test.go`

**Steps:**

- [ ] NotificationService 구현 — Notification 레코드 생성 + 채널별 디스패치

```go
// backend/internal/service/notification_service.go
package service

import (
	"context"
	"log/slog"

	"nexus/backend/internal/infra/sse"
	"nexus/backend/internal/infra/slack"
	"nexus/backend/internal/infra/telegram"
	"nexus/backend/internal/infra/webpush"
	"nexus/backend/internal/repository"
	db "nexus/backend/internal/repository/sqlc"
)

// NotificationPayload is the input for creating a notification.
type NotificationPayload struct {
	UserID      string
	CaseID      *string
	Type        string // matches notification_type enum
	Title       string
	Body        string
	Data        map[string]interface{}
	ImpactScore *int
}

// NotificationService handles notification creation and multi-channel dispatch.
type NotificationService struct {
	repo       *repository.NotificationRepo
	rules      *NotificationRulesEngine
	sseManager *sse.Manager
	slackSvc   *slack.WebhookSender
	tgBot      *telegram.Bot
	pushSender *webpush.Sender
	logger     *slog.Logger
}

func NewNotificationService(
	repo *repository.NotificationRepo,
	rules *NotificationRulesEngine,
	sseMgr *sse.Manager,
	slackSvc *slack.WebhookSender,
	tgBot *telegram.Bot,
	pushSender *webpush.Sender,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		repo:       repo,
		rules:      rules,
		sseManager: sseMgr,
		slackSvc:   slackSvc,
		tgBot:      tgBot,
		pushSender: pushSender,
		logger:     logger,
	}
}

// Emit evaluates rules, creates a Notification record, and dispatches to active channels.
func (s *NotificationService) Emit(ctx context.Context, payload NotificationPayload) error {
	// 1. 사용자 알림 설정 조회
	prefs, err := s.repo.GetPreferences(ctx, payload.UserID)
	if err != nil {
		// 설정이 없으면 기본값 사용 (inApp만 활성)
		s.logger.Warn("notification preferences not found, using defaults",
			"userId", payload.UserID, "error", err)
		prefs = db.NotificationPreference{}
	}

	// 2. 규칙 엔진 평가
	result := s.rules.Evaluate(payload, prefs)
	if !result.ShouldNotify {
		s.logger.Debug("notification filtered by rules",
			"userId", payload.UserID, "reason", result.Reason)
		return nil
	}

	// 3. Notification 레코드 생성
	notification, err := s.repo.Create(ctx, repository.CreateNotificationParams{
		UserID: payload.UserID,
		CaseID: payload.CaseID,
		Type:   payload.Type,
		Title:  payload.Title,
		Body:   payload.Body,
		Data:   payload.Data,
	})
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// 4. 채널별 디스패치 (비동기, 실패해도 다른 채널에 영향 없음)
	s.dispatch(ctx, notification, prefs, result.Channels)

	return nil
}

// dispatch sends to each active channel concurrently.
func (s *NotificationService) dispatch(
	ctx context.Context,
	n db.Notification,
	prefs db.NotificationPreference,
	channels ChannelFlags,
) {
	// In-App (SSE) — 항상 시도
	if channels.InApp {
		s.sseManager.SendToUser(n.UserID.String(), sse.Event{
			Type: "notification",
			Data: map[string]interface{}{
				"id":        n.ID.String(),
				"type":      string(n.Type),
				"title":     n.Title,
				"body":      n.Body,
				"caseId":    nullUUIDToString(n.CaseID),
				"createdAt": n.CreatedAt.Format(time.RFC3339),
			},
		})
	}

	// Web Push
	if channels.Push {
		go func() {
			subs, err := s.repo.ListPushSubscriptions(ctx, n.UserID.String())
			if err != nil {
				s.logger.Error("failed to list push subscriptions", "error", err)
				return
			}
			expiredIDs, err := s.pushSender.Send(n, subs)
			if err != nil {
				s.logger.Error("push send error", "error", err)
			}
			// 만료된 구독 삭제
			if len(expiredIDs) > 0 {
				if err := s.repo.DeletePushSubscriptionsByIDs(ctx, expiredIDs); err != nil {
					s.logger.Error("failed to delete expired subscriptions", "error", err)
				}
			}
		}()
	}

	// Slack
	if channels.Slack && prefs.SlackWebhookUrl != nil {
		go func() {
			if err := s.slackSvc.Send(n, *prefs.SlackWebhookUrl); err != nil {
				s.logger.Error("slack send error", "error", err)
			}
		}()
	}

	// Telegram
	if channels.Telegram && prefs.TelegramChatId != nil {
		go func() {
			if err := s.tgBot.SendMessage(*prefs.TelegramChatId, n); err != nil {
				s.logger.Error("telegram send error", "error", err)
			}
		}()
	}
}

func nullUUIDToString(u uuid.NullUUID) *string {
	if !u.Valid {
		return nil
	}
	s := u.UUID.String()
	return &s
}
```

- [ ] Plan 6 lifecycle-handler의 `emitNotification` 호출을 `NotificationService.Emit()`으로 변경
- [ ] 테스트: Emit이 muted 케이스/타입을 필터링하는지 확인
- [ ] 테스트: impactScore 미달 시 알림 스킵 확인
- [ ] 테스트: 디스패처가 활성 채널에만 전달하는지 확인

```bash
cd backend && go test ./internal/service/ -run TestNotificationService -v
git add backend/internal/service/notification_service.go
git commit -m "feat(alerts): NotificationService 및 채널 디스패처 구현 (Go)"
```

---

## Task 3: SSE 실시간 스트림 (In-App 알림)

브라우저에서 실시간 알림을 수신하기 위한 SSE Manager와 Gin SSE 핸들러를 구현한다.

**Files:**
- Create: `backend/internal/infra/sse/manager.go`
- Create: `backend/internal/infra/sse/manager_test.go`
- Modify: `backend/internal/handler/notification_handler.go` (Stream 메서드 추가)

**Steps:**

- [ ] SSE Manager 구현 — 사용자별 연결 풀 관리, heartbeat

```go
// backend/internal/infra/sse/manager.go
package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Event represents an SSE event to send.
type Event struct {
	Type string
	Data interface{}
}

// Manager manages per-user SSE connections.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]map[chan Event]struct{} // userID -> set of channels
	logger      *slog.Logger
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		connections: make(map[string]map[chan Event]struct{}),
		logger:      logger,
	}
}

// Subscribe registers a new SSE channel for the given user.
func (m *Manager) Subscribe(userID string) chan Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan Event, 64) // buffered to prevent blocking
	if m.connections[userID] == nil {
		m.connections[userID] = make(map[chan Event]struct{})
	}
	m.connections[userID][ch] = struct{}{}

	m.logger.Debug("SSE client subscribed", "userId", userID, "total", len(m.connections[userID]))
	return ch
}

// Unsubscribe removes an SSE channel for the given user.
func (m *Manager) Unsubscribe(userID string, ch chan Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conns, ok := m.connections[userID]; ok {
		delete(conns, ch)
		close(ch)
		if len(conns) == 0 {
			delete(m.connections, userID)
		}
	}

	m.logger.Debug("SSE client unsubscribed", "userId", userID)
}

// SendToUser sends an event to all SSE connections of a user.
func (m *Manager) SendToUser(userID string, event Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns, ok := m.connections[userID]
	if !ok {
		return
	}

	for ch := range conns {
		select {
		case ch <- event:
		default:
			// channel full, skip to prevent blocking
			m.logger.Warn("SSE channel full, dropping event", "userId", userID)
		}
	}
}

// ConnectionCount returns the number of active SSE connections for a user.
func (m *Manager) ConnectionCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections[userID])
}

// TotalConnections returns the total number of active SSE connections.
func (m *Manager) TotalConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := 0
	for _, conns := range m.connections {
		total += len(conns)
	}
	return total
}

// StartHeartbeat sends periodic heartbeat comments to all connections.
// Call this in a goroutine: go manager.StartHeartbeat(ctx)
func (m *Manager) StartHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			for userID, conns := range m.connections {
				for ch := range conns {
					select {
					case ch <- Event{Type: "heartbeat", Data: nil}:
					default:
						m.logger.Warn("heartbeat dropped", "userId", userID)
					}
				}
			}
			m.mu.RUnlock()
		}
	}
}

// FormatSSE formats an Event into SSE wire format.
func FormatSSE(event Event) string {
	if event.Type == "heartbeat" {
		return ": heartbeat\n\n"
	}

	dataBytes, err := json.Marshal(event.Data)
	if err != nil {
		dataBytes = []byte("{}")
	}

	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(dataBytes))
}
```

- [ ] SSE Stream 핸들러 — Gin의 c.Stream() 사용

```go
// backend/internal/handler/notification_handler.go (Stream 메서드 추가)

// GET /api/notifications/stream — SSE endpoint
func (h *NotificationHandler) Stream(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// SSE 헤더 설정
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx proxy 버퍼링 비활성화

	// 연결 등록
	ch := h.sseManager.Subscribe(userID)
	defer h.sseManager.Unsubscribe(userID, ch)

	// 연결 확인 메시지
	c.Writer.WriteString(
		sse.FormatSSE(sse.Event{
			Type: "connected",
			Data: map[string]string{"userId": userID},
		}),
	)
	c.Writer.Flush()

	// 이벤트 스트리밍
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			c.Writer.WriteString(sse.FormatSSE(event))
			c.Writer.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
```

- [ ] 30초 간격 heartbeat — 서버 시작 시 `go sseManager.StartHeartbeat(ctx)` 호출
- [ ] 테스트: SSE 연결 수립 -> 알림 전송 -> 클라이언트 수신 확인
- [ ] 테스트: 다중 탭(동일 userId 복수 연결) 시 모든 탭에 전달 확인
- [ ] 테스트: 연결 종료 시 연결 풀에서 제거 확인

```bash
cd backend && go test ./internal/infra/sse/ -v
git add backend/internal/infra/sse/ backend/internal/handler/notification_handler.go
git commit -m "feat(alerts): SSE 실시간 알림 스트림 및 연결 관리자 (Go Gin)"
```

---

## Task 4: Web Push Notification (VAPID)

브라우저 Push Notification을 위한 VAPID 키 관리, 구독 등록, 푸시 전송을 Go로 구현한다.

**Files:**
- Create: `backend/internal/infra/webpush/sender.go`
- Create: `backend/internal/infra/webpush/sender_test.go`
- Modify: `backend/internal/handler/notification_handler.go` (PushSubscribe, VAPIDKey 메서드 추가)
- Create: `public/sw.js` (Service Worker — 프론트엔드, 유지)

**Steps:**

- [ ] Go web push 라이브러리 추가

```bash
cd backend && go get github.com/SherClockHolmes/webpush-go
```

- [ ] 환경 변수에 VAPID 키 추가

```env
VAPID_PUBLIC_KEY=BPxxxxxxxx
VAPID_PRIVATE_KEY=xxxxxxxx
VAPID_MAILTO=mailto:admin@nexus.app
```

- [ ] Web Push Sender 구현

```go
// backend/internal/infra/webpush/sender.go
package webpush

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	wpush "github.com/SherClockHolmes/webpush-go"
	db "nexus/backend/internal/repository/sqlc"
)

// Config holds VAPID configuration.
type Config struct {
	PublicKey  string
	PrivateKey string
	Mailto    string
}

// Sender sends Web Push notifications using VAPID.
type Sender struct {
	config Config
	logger *slog.Logger
}

func NewSender(cfg Config, logger *slog.Logger) *Sender {
	return &Sender{config: cfg, logger: logger}
}

// PushPayload is the JSON payload sent to the browser.
type PushPayload struct {
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Icon  string                 `json:"icon"`
	Badge string                 `json:"badge"`
	Data  map[string]interface{} `json:"data"`
}

// Send pushes a notification to all provided subscriptions.
// Returns IDs of subscriptions that returned 410 Gone (expired).
func (s *Sender) Send(n db.Notification, subscriptions []db.PushSubscription) ([]string, error) {
	payload := PushPayload{
		Title: n.Title,
		Body:  n.Body,
		Icon:  "/icons/nexus-192.png",
		Badge: "/icons/nexus-badge.png",
		Data: map[string]interface{}{
			"notificationId": n.ID.String(),
			"caseId":         nullUUIDToString(n.CaseID),
			"type":           string(n.Type),
			"url":            buildURL(n),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal push payload: %w", err)
	}

	var expiredIDs []string

	for _, sub := range subscriptions {
		subscription := &wpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: wpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := wpush.SendNotification(payloadBytes, subscription, &wpush.Options{
			Subscriber:      s.config.Mailto,
			VAPIDPublicKey:  s.config.PublicKey,
			VAPIDPrivateKey: s.config.PrivateKey,
			TTL:             60,
		})
		if err != nil {
			s.logger.Error("web push send failed", "endpoint", sub.Endpoint, "error", err)
			continue
		}
		resp.Body.Close()

		// 410 Gone = 구독 만료
		if resp.StatusCode == http.StatusGone {
			expiredIDs = append(expiredIDs, sub.ID.String())
			s.logger.Info("push subscription expired", "id", sub.ID.String())
		}
	}

	return expiredIDs, nil
}

func buildURL(n db.Notification) string {
	if n.CaseID.Valid {
		return fmt.Sprintf("/cases/%s", n.CaseID.UUID.String())
	}
	return "/notifications"
}

func nullUUIDToString(u interface{ Valid bool; UUID interface{ String() string } }) *string {
	// helper — 실제 구현에서는 uuid.NullUUID 사용
	return nil
}
```

- [ ] Push 구독 등록 핸들러

```go
// backend/internal/handler/notification_handler.go (PushSubscribe, VAPIDKey 추가)

// POST /api/notifications/push/subscribe
func (h *NotificationHandler) PushSubscribe(c *gin.Context) {
	userID := c.GetString("userId")

	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
		Keys     struct {
			P256dh string `json:"p256dh" binding:"required"`
			Auth   string `json:"auth" binding:"required"`
		} `json:"keys" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.repo.CreatePushSubscription(c.Request.Context(), repository.CreatePushSubParams{
		UserID:   userID,
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": sub.ID.String()})
}

// GET /api/notifications/push/vapid-key
func (h *NotificationHandler) VAPIDKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"publicKey": h.pushSender.Config().PublicKey,
	})
}
```

- [ ] Service Worker (프론트엔드 유지)

```javascript
// public/sw.js (프론트엔드 — 변경 없음)
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? {};
  event.waitUntil(
    self.registration.showNotification(data.title || 'NEXUS', {
      body: data.body,
      icon: data.icon || '/icons/nexus-192.png',
      badge: data.badge || '/icons/nexus-badge.png',
      data: data.data,
    })
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = event.notification.data?.url || '/';
  event.waitUntil(clients.openWindow(url));
});
```

- [ ] 테스트: 구독 등록 -> 푸시 전송 -> webpush 호출 확인
- [ ] 테스트: 410 응답 시 구독 자동 삭제 확인

```bash
cd backend && go test ./internal/infra/webpush/ -v
git add backend/internal/infra/webpush/ backend/internal/handler/notification_handler.go public/sw.js
git commit -m "feat(alerts): Web Push Notification VAPID (Go webpush-go)"
```

---

## Task 5: Slack / Telegram 채널 연동

Slack Incoming Webhook과 Telegram Bot API를 통한 외부 메신저 알림 채널을 Go로 구현한다.

**Files:**
- Create: `backend/internal/infra/slack/webhook.go`
- Create: `backend/internal/infra/slack/webhook_test.go`
- Create: `backend/internal/infra/telegram/bot.go`
- Create: `backend/internal/infra/telegram/bot_test.go`
- Modify: `backend/internal/handler/notification_handler.go` (GetSettings, UpdateSettings 추가)

**Steps:**

- [ ] Slack Webhook Sender 구현 — Block Kit 메시지 전송

```go
// backend/internal/infra/slack/webhook.go
package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	db "nexus/backend/internal/repository/sqlc"
)

// WebhookSender sends notifications via Slack Incoming Webhook.
type WebhookSender struct {
	client *http.Client
	logger *slog.Logger
}

func NewWebhookSender(logger *slog.Logger) *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

var typeEmoji = map[db.NotificationType]string{
	"PRICE_ALERT":       ":chart_with_upwards_trend:",
	"SUCCESS_CONDITION":  ":white_check_mark:",
	"FAILURE_CONDITION":  ":x:",
	"NEWS_EVENT":         ":newspaper:",
	"DISCLOSURE_EVENT":   ":page_facing_up:",
	"SECTOR_EVENT":       ":bar_chart:",
	"MONITOR_RESULT":     ":robot_face:",
}

// Send posts a Block Kit message to the given Slack webhook URL.
func (s *WebhookSender) Send(n db.Notification, webhookURL string) error {
	emoji := typeEmoji[n.Type]
	if emoji == "" {
		emoji = ":bell:"
	}

	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": fmt.Sprintf("%s %s", emoji, n.Title),
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": n.Body,
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("NEXUS | %s", n.CreatedAt.Format("2006-01-02 15:04:05")),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := s.client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] Telegram Bot 구현 — Bot API sendMessage, HTML parse mode

```go
// backend/internal/infra/telegram/bot.go
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"time"

	db "nexus/backend/internal/repository/sqlc"
)

// Bot sends notifications via Telegram Bot API.
type Bot struct {
	token  string
	client *http.Client
	logger *slog.Logger
}

func NewBot(token string, logger *slog.Logger) *Bot {
	return &Bot{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

var typeLabel = map[db.NotificationType]string{
	"PRICE_ALERT":       "📈 가격 알림",
	"SUCCESS_CONDITION":  "✅ 성공 조건 도달",
	"FAILURE_CONDITION":  "❌ 실패 조건 도달",
	"NEWS_EVENT":         "📰 뉴스",
	"DISCLOSURE_EVENT":   "📄 공시",
	"SECTOR_EVENT":       "📊 섹터",
	"MONITOR_RESULT":     "🤖 모니터링",
}

// SendMessage sends a formatted notification to a Telegram chat.
func (b *Bot) SendMessage(chatID string, n db.Notification) error {
	label := typeLabel[n.Type]
	if label == "" {
		label = "🔔 알림"
	}

	text := fmt.Sprintf(
		"<b>%s</b>\n<b>%s</b>\n\n%s\n\n<i>NEXUS | %s</i>",
		html.EscapeString(label),
		html.EscapeString(n.Title),
		html.EscapeString(n.Body),
		n.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := b.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] 알림 설정 핸들러 — GetSettings / UpdateSettings

```go
// backend/internal/handler/notification_handler.go (Settings 메서드 추가)

// GET /api/notifications/settings
func (h *NotificationHandler) GetSettings(c *gin.Context) {
	userID := c.GetString("userId")

	prefs, err := h.repo.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		// 설정이 없으면 기본값 반환
		c.JSON(http.StatusOK, gin.H{
			"inAppEnabled":    true,
			"pushEnabled":     false,
			"slackEnabled":    false,
			"telegramEnabled": false,
			"minImpactScore":  5,
			"mutedCaseIds":    []string{},
			"mutedTypes":      []string{},
		})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// PUT /api/notifications/settings
func (h *NotificationHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("userId")

	var req struct {
		InAppEnabled    bool     `json:"inAppEnabled"`
		PushEnabled     bool     `json:"pushEnabled"`
		SlackEnabled    bool     `json:"slackEnabled"`
		TelegramEnabled bool     `json:"telegramEnabled"`
		SlackWebhookUrl *string  `json:"slackWebhookUrl"`
		TelegramChatId  *string  `json:"telegramChatId"`
		MinImpactScore  int      `json:"minImpactScore"`
		MutedCaseIds    []string `json:"mutedCaseIds"`
		MutedTypes      []string `json:"mutedTypes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := h.repo.UpsertPreferences(c.Request.Context(), repository.UpsertPrefsParams{
		UserID:          userID,
		InAppEnabled:    req.InAppEnabled,
		PushEnabled:     req.PushEnabled,
		SlackEnabled:    req.SlackEnabled,
		TelegramEnabled: req.TelegramEnabled,
		SlackWebhookUrl: req.SlackWebhookUrl,
		TelegramChatId:  req.TelegramChatId,
		MinImpactScore:  req.MinImpactScore,
		MutedCaseIds:    req.MutedCaseIds,
		MutedTypes:      req.MutedTypes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
```

- [ ] `.env.example`에 `TELEGRAM_BOT_TOKEN` 추가
- [ ] 테스트: Slack webhook 호출 시 Block Kit 형식 확인
- [ ] 테스트: Telegram sendMessage 호출 시 HTML 이스케이프 확인
- [ ] 테스트: 채널 비활성화 시 해당 채널 전송 스킵 확인

```bash
cd backend && go test ./internal/infra/slack/ ./internal/infra/telegram/ -v
git add backend/internal/infra/slack/ backend/internal/infra/telegram/ backend/internal/handler/
git commit -m "feat(alerts): Slack / Telegram 채널 연동 (Go)"
```

---

## Task 6: 알림 규칙 엔진 및 통합 테스트

케이스별/타입별/임팩트 기준 알림 필터링 규칙을 정교화하고, 전체 알림 파이프라인을 통합 테스트한다.

**Files:**
- Create: `backend/internal/service/notification_rules.go`
- Create: `backend/internal/service/notification_rules_test.go`
- Create: `backend/internal/service/notification_integration_test.go`

**Steps:**

- [ ] 알림 규칙 엔진 구현 — 복합 필터링 로직 분리

```go
// backend/internal/service/notification_rules.go
package service

import (
	db "nexus/backend/internal/repository/sqlc"
)

// ChannelFlags indicates which channels are active.
type ChannelFlags struct {
	InApp    bool
	Push     bool
	Slack    bool
	Telegram bool
}

// RuleResult is the output of the rules engine evaluation.
type RuleResult struct {
	ShouldNotify bool
	Reason       string
	Channels     ChannelFlags
}

// NotificationRulesEngine evaluates whether a notification should be sent.
type NotificationRulesEngine struct{}

func NewNotificationRulesEngine() *NotificationRulesEngine {
	return &NotificationRulesEngine{}
}

// alwaysNotifyTypes are notification types that bypass impact score filtering.
var alwaysNotifyTypes = map[string]bool{
	"SUCCESS_CONDITION": true,
	"FAILURE_CONDITION": true,
	"PRICE_ALERT":       true,
}

// Evaluate checks the payload against user preferences and returns a RuleResult.
func (e *NotificationRulesEngine) Evaluate(
	payload NotificationPayload,
	prefs db.NotificationPreference,
) RuleResult {
	defaultChannels := ChannelFlags{
		InApp: true, Push: false, Slack: false, Telegram: false,
	}

	// 설정이 없으면 기본값 (inApp만 활성)
	if prefs.ID.String() == "00000000-0000-0000-0000-000000000000" {
		return RuleResult{ShouldNotify: true, Channels: defaultChannels}
	}

	// 케이스 음소거 체크
	if payload.CaseID != nil {
		for _, mutedID := range prefs.MutedCaseIds {
			if mutedID.String() == *payload.CaseID {
				return RuleResult{
					ShouldNotify: false,
					Reason:       "Case muted",
					Channels:     defaultChannels,
				}
			}
		}
	}

	// 타입 음소거 체크
	for _, mutedType := range prefs.MutedTypes {
		if string(mutedType) == payload.Type {
			return RuleResult{
				ShouldNotify: false,
				Reason:       "Type muted",
				Channels:     defaultChannels,
			}
		}
	}

	// 임팩트 점수 체크 (성공/실패/가격 알림은 항상 통과)
	if !alwaysNotifyTypes[payload.Type] &&
		payload.ImpactScore != nil &&
		*payload.ImpactScore < prefs.MinImpactScore {
		return RuleResult{
			ShouldNotify: false,
			Reason:       "Impact score below threshold",
			Channels:     defaultChannels,
		}
	}

	return RuleResult{
		ShouldNotify: true,
		Channels: ChannelFlags{
			InApp:    prefs.InAppEnabled,
			Push:     prefs.PushEnabled,
			Slack:    prefs.SlackEnabled,
			Telegram: prefs.TelegramEnabled,
		},
	}
}
```

- [ ] 규칙 엔진 단위 테스트

```go
// backend/internal/service/notification_rules_test.go
package service

import (
	"testing"

	"github.com/google/uuid"
	db "nexus/backend/internal/repository/sqlc"
)

func TestEvaluateRules_MutedCase(t *testing.T) {
	engine := NewNotificationRulesEngine()
	caseID := uuid.New().String()
	caseUUID := uuid.MustParse(caseID)

	prefs := db.NotificationPreference{
		ID:           uuid.New(),
		InAppEnabled: true,
		MutedCaseIds: []uuid.UUID{caseUUID},
	}

	result := engine.Evaluate(NotificationPayload{
		UserID: "user1",
		CaseID: &caseID,
		Type:   "NEWS_EVENT",
	}, prefs)

	if result.ShouldNotify {
		t.Error("expected notification to be filtered (muted case)")
	}
	if result.Reason != "Case muted" {
		t.Errorf("expected reason 'Case muted', got '%s'", result.Reason)
	}
}

func TestEvaluateRules_MutedType(t *testing.T) {
	engine := NewNotificationRulesEngine()

	prefs := db.NotificationPreference{
		ID:           uuid.New(),
		InAppEnabled: true,
		MutedTypes:   []db.NotificationType{"SECTOR_EVENT"},
	}

	result := engine.Evaluate(NotificationPayload{
		UserID: "user1",
		Type:   "SECTOR_EVENT",
	}, prefs)

	if result.ShouldNotify {
		t.Error("expected notification to be filtered (muted type)")
	}
}

func TestEvaluateRules_ImpactScoreBelowThreshold(t *testing.T) {
	engine := NewNotificationRulesEngine()
	score := 3

	prefs := db.NotificationPreference{
		ID:             uuid.New(),
		InAppEnabled:   true,
		MinImpactScore: 5,
	}

	result := engine.Evaluate(NotificationPayload{
		UserID:      "user1",
		Type:        "NEWS_EVENT",
		ImpactScore: &score,
	}, prefs)

	if result.ShouldNotify {
		t.Error("expected notification to be filtered (low impact score)")
	}
}

func TestEvaluateRules_SuccessConditionBypassesImpactScore(t *testing.T) {
	engine := NewNotificationRulesEngine()
	score := 1

	prefs := db.NotificationPreference{
		ID:             uuid.New(),
		InAppEnabled:   true,
		MinImpactScore: 10,
	}

	result := engine.Evaluate(NotificationPayload{
		UserID:      "user1",
		Type:        "SUCCESS_CONDITION",
		ImpactScore: &score,
	}, prefs)

	if !result.ShouldNotify {
		t.Error("SUCCESS_CONDITION should always notify regardless of impact score")
	}
}

func TestEvaluateRules_ChannelFlags(t *testing.T) {
	engine := NewNotificationRulesEngine()

	prefs := db.NotificationPreference{
		ID:              uuid.New(),
		InAppEnabled:    true,
		PushEnabled:     true,
		SlackEnabled:    true,
		TelegramEnabled: false,
	}

	result := engine.Evaluate(NotificationPayload{
		UserID: "user1",
		Type:   "PRICE_ALERT",
	}, prefs)

	if !result.ShouldNotify {
		t.Error("expected notification to be sent")
	}
	if !result.Channels.InApp || !result.Channels.Push || !result.Channels.Slack {
		t.Error("expected InApp, Push, Slack to be enabled")
	}
	if result.Channels.Telegram {
		t.Error("expected Telegram to be disabled")
	}
}
```

- [ ] 통합 테스트 — 전체 알림 파이프라인 검증

```go
// backend/internal/service/notification_integration_test.go
package service

import (
	"context"
	"testing"
	// ... mock 패키지
)

func TestIntegration_EmitToSSE(t *testing.T) {
	// 1. SSE Manager에 연결 mock
	// 2. NotificationService.Emit 호출 (impactScore=8)
	// 3. Notification 레코드 생성 확인
	// 4. SSE 채널로 이벤트 전송 확인
}

func TestIntegration_LowImpactScoreSkipped(t *testing.T) {
	// prefs.MinImpactScore=7, payload.ImpactScore=5
	// Notification 레코드 생성 안 됨
}

func TestIntegration_SuccessConditionAlwaysNotifies(t *testing.T) {
	// type=SUCCESS_CONDITION, impactScore 없음 → 알림 전송
}

func TestIntegration_MutedCaseSkipsAllChannels(t *testing.T) {
	// mutedCaseIds에 해당 caseId 포함
	// 어떤 채널도 호출 안 됨
}

func TestIntegration_MultiChannelDispatch(t *testing.T) {
	// 모든 채널 enabled
	// In-App + Slack + Telegram 각각 전송 확인
}
```

- [ ] 테스트 실행 및 전체 통과 확인

```bash
cd backend && go test ./internal/service/ -v
git add backend/internal/service/
git commit -m "feat(alerts): 알림 규칙 엔진 및 통합 테스트 완성 (Go)"
```

---

## 프론트엔드 변경 사항 (API 호출 경로만 변경)

기존 `src/features/notification/` UI 컴포넌트는 유지하되, API 호출을 Go 백엔드로 변경한다.

```typescript
// src/features/notification/api/notification.api.ts
// 기존: /api/notifications → 변경: ${GO_BACKEND_URL}/api/notifications

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function fetchNotifications(params: { page?: number; limit?: number; unreadOnly?: boolean }) {
  const url = new URL(`${API_BASE}/api/notifications`);
  if (params.page) url.searchParams.set('page', String(params.page));
  if (params.limit) url.searchParams.set('limit', String(params.limit));
  if (params.unreadOnly) url.searchParams.set('unreadOnly', 'true');
  const res = await fetch(url.toString(), { credentials: 'include' });
  return res.json();
}

export async function markAsRead(id: string) {
  const res = await fetch(`${API_BASE}/api/notifications/${id}/read`, {
    method: 'PUT', credentials: 'include',
  });
  return res.json();
}

export function connectSSE(onEvent: (event: MessageEvent) => void): EventSource {
  const es = new EventSource(`${API_BASE}/api/notifications/stream`, { withCredentials: true });
  es.addEventListener('notification', onEvent);
  return es;
}

export async function fetchUnreadCount(): Promise<number> {
  const res = await fetch(`${API_BASE}/api/notifications/unread-count`, { credentials: 'include' });
  const data = await res.json();
  return data.count;
}

export async function updateSettings(settings: Record<string, unknown>) {
  const res = await fetch(`${API_BASE}/api/notifications/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
    credentials: 'include',
  });
  return res.json();
}
```

---

## 파일 구조 요약

### Go 백엔드 (신규)
```
backend/internal/
  handler/
    notification_handler.go       # Gin handlers: List, MarkAsRead, Stream, Settings, Push
    notification_handler_test.go
  service/
    notification_service.go       # Emit + dispatch (emitter + dispatcher 통합)
    notification_service_test.go
    notification_rules.go         # 규칙 엔진 (필터링)
    notification_rules_test.go
    notification_integration_test.go
  repository/
    notification_repo.go          # sqlc CRUD 래퍼
    sqlc/
      migrations/007_add_notification_models.sql
      queries/notification.sql
  infra/
    sse/
      manager.go                  # SSE 연결 풀 관리 + heartbeat
      manager_test.go
    slack/
      webhook.go                  # Slack Block Kit webhook
      webhook_test.go
    telegram/
      bot.go                      # Telegram Bot API (HTML)
      bot_test.go
    webpush/
      sender.go                   # Web Push VAPID (webpush-go)
      sender_test.go
```

### 프론트엔드 (유지, API 경로만 변경)
```
src/features/notification/
  api/notification.api.ts         # API 호출 (Go 백엔드 URL로 변경)
  ui/                             # 알림 목록, 설정 패널 UI (변경 없음)
public/
  sw.js                           # Service Worker (변경 없음)
```
