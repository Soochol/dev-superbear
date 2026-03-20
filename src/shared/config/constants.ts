// ---------------------------------------------------------------------------
// NEXUS — Global Constants
// ---------------------------------------------------------------------------

/** Application metadata */
export const APP_NAME = "NEXUS" as const;
export const APP_DESCRIPTION = "AI-Native Investment Intelligence" as const;

/** Backend API base URL (set via environment variable in production) */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

export interface NavItem {
  label: string;
  href: string;
  icon?: string;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { label: "Dashboard", href: "/", icon: "layout-dashboard" },
  { label: "Cases", href: "/cases", icon: "briefcase" },
  { label: "Pipelines", href: "/pipelines", icon: "git-branch" },
  { label: "Trades", href: "/trades", icon: "trending-up" },
  { label: "Alerts", href: "/alerts", icon: "bell" },
  { label: "Timeline", href: "/timeline", icon: "clock" },
] as const;

// ---------------------------------------------------------------------------
// Case / Pipeline status enums
// ---------------------------------------------------------------------------

export const CASE_STATUS = {
  LIVE: "LIVE",
  CLOSED_SUCCESS: "CLOSED_SUCCESS",
  CLOSED_FAILURE: "CLOSED_FAILURE",
  BACKTEST: "BACKTEST",
} as const;

export type CaseStatusType = (typeof CASE_STATUS)[keyof typeof CASE_STATUS];

export const PipelineStatus = {
  IDLE: "idle",
  RUNNING: "running",
  SUCCESS: "success",
  FAILED: "failed",
  CANCELLED: "cancelled",
} as const;

export type PipelineStatusType =
  (typeof PipelineStatus)[keyof typeof PipelineStatus];

// ---------------------------------------------------------------------------
// Agent block execution status
// ---------------------------------------------------------------------------

export const BlockStatus = {
  PENDING: "pending",
  RUNNING: "running",
  COMPLETED: "completed",
  ERROR: "error",
  SKIPPED: "skipped",
} as const;

export type BlockStatusType = (typeof BlockStatus)[keyof typeof BlockStatus];

// ---------------------------------------------------------------------------
// Trade direction
// ---------------------------------------------------------------------------

export const TRADE_TYPE = {
  BUY: "BUY",
  SELL: "SELL",
} as const;

export type TradeTypeValue =
  (typeof TRADE_TYPE)[keyof typeof TRADE_TYPE];

// ---------------------------------------------------------------------------
// Alert severity
// ---------------------------------------------------------------------------

export const AlertSeverity = {
  INFO: "info",
  WARNING: "warning",
  CRITICAL: "critical",
} as const;

export type AlertSeverityType =
  (typeof AlertSeverity)[keyof typeof AlertSeverity];

// ---------------------------------------------------------------------------
// Pagination defaults
// ---------------------------------------------------------------------------

export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;
