# Sidebar Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 현재 상단 네비게이션 바를 접이식 사이드바(64px ↔ 200px, 핀 모드)로 전환하고, 프로토타입의 9개 페이지를 `(app)` route group으로 통합한다.

**Architecture:** `(app)` route group에 `SidebarLayout` 클라이언트 컴포넌트를 배치. 사이드바 상태(확장/핀)는 Zustand store로 관리하며, 핀 상태는 localStorage에 영속. 기존 4개 페이지는 디렉토리 이동만, 5개 신규 페이지는 placeholder로 생성.

**Tech Stack:** Next.js 16, React 19, Tailwind CSS v4, Zustand 5

**Spec:** `docs/superpowers/specs/2026-03-20-sidebar-navigation-design.md`

**IMPORTANT:** Next.js 16은 기존 버전과 다를 수 있음. 코드 작성 전 `node_modules/next/dist/docs/`의 관련 문서를 확인할 것.

**Spec deviation:** 스펙의 `SidebarSearch.tsx`는 별도 컴포넌트로 명시되어 있으나, `/search` 네비게이션 링크에 불과하므로 `SidebarNavItem`으로 통합 처리한다. 별도 컴포넌트는 불필요한 추상화.

**E2E 영향 분석:** 기존 E2E 테스트(chart-page, search, pipeline-builder)는 URL 직접 접근(`page.goto`)을 사용하며, `AppNavigation` 컴포넌트에 의존하지 않음. Route group 변경(`(pages)` → `(app)`)은 URL 경로에 영향 없으므로 기존 E2E 테스트 수정 불필요. 단, `e2e/landing.spec.ts`는 루트 redirect로 인해 업데이트 필요.

---

## File Structure

```
src/
├── app/
│   ├── layout.tsx                          ← MODIFY: flex wrapper 제거
│   ├── page.tsx                            ← MODIFY: /dashboard redirect
│   ├── globals.css                         ← MODIFY: 사이드바 토큰 추가
│   ├── (app)/                              ← CREATE: route group
│   │   ├── layout.tsx                      ← CREATE: SidebarLayout 사용
│   │   ├── dashboard/page.tsx              ← CREATE: placeholder
│   │   ├── search/page.tsx                 ← MOVE from (pages)/search/
│   │   ├── chart/                          ← MOVE from (pages)/chart/
│   │   │   ├── layout.tsx
│   │   │   └── page.tsx
│   │   ├── pipeline/                       ← MOVE from pipeline/
│   │   │   ├── layout.tsx
│   │   │   └── page.tsx
│   │   ├── cases/                          ← MOVE from (pages)/cases/
│   │   │   ├── page.tsx
│   │   │   └── [id]/page.tsx
│   │   ├── backtest/page.tsx               ← CREATE: placeholder
│   │   ├── portfolio/page.tsx              ← CREATE: placeholder
│   │   ├── alerts/page.tsx                 ← CREATE: placeholder
│   │   └── marketplace/page.tsx            ← CREATE: placeholder
│   ├── (pages)/                            ← DELETE after migration
│   └── pipeline/                           ← DELETE after migration
├── widgets/
│   └── app-sidebar/                        ← CREATE: new widget
│       ├── ui/
│       │   ├── AppSidebar.tsx
│       │   ├── SidebarLayout.tsx
│       │   ├── SidebarLogo.tsx
│       │   ├── SidebarNavItem.tsx
│       │   └── SidebarUserInfo.tsx
│       ├── __tests__/
│       │   └── AppSidebar.test.tsx
│       └── index.ts
├── shared/
│   ├── model/
│   │   └── sidebar.store.ts               ← CREATE: 사이드바 UI 상태
│   ├── model/__tests__/
│   │   └── sidebar.store.test.ts           ← CREATE: store 테스트
│   └── ui/
│       └── AppNavigation.tsx               ← DELETE
```

---

### Task 1: Design Tokens

**Files:**
- Modify: `src/app/globals.css`

- [ ] **Step 1: Add sidebar color tokens**

`src/app/globals.css` — `@theme inline` 블록 안에 추가:

```css
  --color-nexus-sidebar: #0e0e16;
  --color-nexus-sidebar-hover: #1a1a28;
  --color-nexus-sidebar-active: rgba(99, 102, 241, 0.1);
```

기존 `--color-nexus-text-muted` 줄 다음에 삽입.

- [ ] **Step 2: Commit**

```bash
git add src/app/globals.css
git commit -m "style: add sidebar design tokens"
```

---

### Task 2: Sidebar Zustand Store (TDD)

**Files:**
- Create: `src/shared/model/sidebar.store.ts`
- Test: `src/shared/model/__tests__/sidebar.store.test.ts`

- [ ] **Step 1: Write failing tests**

```typescript
// src/shared/model/__tests__/sidebar.store.test.ts
import { useSidebarStore } from "@/shared/model/sidebar.store";

describe("sidebarStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarStore.setState(useSidebarStore.getInitialState());
  });

  it("starts collapsed and unpinned", () => {
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(false);
    expect(state.isExpanded).toBe(false);
  });

  it("setExpanded changes isExpanded when not pinned", () => {
    useSidebarStore.getState().setExpanded(true);
    expect(useSidebarStore.getState().isExpanded).toBe(true);

    useSidebarStore.getState().setExpanded(false);
    expect(useSidebarStore.getState().isExpanded).toBe(false);
  });

  it("setExpanded is ignored when pinned", () => {
    useSidebarStore.setState({ isPinned: true, isExpanded: true });

    useSidebarStore.getState().setExpanded(false);
    expect(useSidebarStore.getState().isExpanded).toBe(true);
  });

  it("togglePin pins and expands", () => {
    useSidebarStore.getState().togglePin();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(true);
    expect(state.isExpanded).toBe(true);
  });

  it("togglePin unpins and collapses", () => {
    useSidebarStore.setState({ isPinned: true, isExpanded: true });

    useSidebarStore.getState().togglePin();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(false);
    expect(state.isExpanded).toBe(false);
  });

  it("persists isPinned to localStorage", () => {
    useSidebarStore.getState().togglePin();
    expect(localStorage.getItem("sidebar-pinned")).toBe("true");

    useSidebarStore.getState().togglePin();
    expect(localStorage.getItem("sidebar-pinned")).toBe("false");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx jest src/shared/model/__tests__/sidebar.store.test.ts --verbose`
Expected: FAIL — module not found

- [ ] **Step 3: Implement sidebar store**

```typescript
// src/shared/model/sidebar.store.ts
import { create } from "zustand";

interface SidebarState {
  isPinned: boolean;
  isExpanded: boolean;
  togglePin: () => void;
  setExpanded: (expanded: boolean) => void;
}

function readPinned(): boolean {
  if (typeof window === "undefined") return false;
  return localStorage.getItem("sidebar-pinned") === "true";
}

export const useSidebarStore = create<SidebarState>((set, get) => ({
  isPinned: readPinned(),
  isExpanded: readPinned(),
  togglePin: () => {
    const next = !get().isPinned;
    set({ isPinned: next, isExpanded: next });
    if (typeof window !== "undefined") {
      localStorage.setItem("sidebar-pinned", String(next));
    }
  },
  setExpanded: (expanded) => {
    if (!get().isPinned) {
      set({ isExpanded: expanded });
    }
  },
}));
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npx jest src/shared/model/__tests__/sidebar.store.test.ts --verbose`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/shared/model/sidebar.store.ts src/shared/model/__tests__/sidebar.store.test.ts
git commit -m "feat: add sidebar zustand store with pin/expand state"
```

---

### Task 3: SidebarNavItem Component (TDD)

**Files:**
- Create: `src/widgets/app-sidebar/ui/SidebarNavItem.tsx`
- Test in: `src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx` (shared test file)

이 컴포넌트는 `usePathname()`을 사용하므로 Next.js mock이 필요하다. 기존 프로젝트의 `src/__mocks__/` 패턴을 확인할 것.

- [ ] **Step 1: Write failing tests**

```tsx
// src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx
/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";

let mockPathname = "/dashboard";
jest.mock("next/navigation", () => ({
  usePathname: () => mockPathname,
}));

import { SidebarNavItem } from "../ui/SidebarNavItem";

describe("SidebarNavItem", () => {
  beforeEach(() => {
    mockPathname = "/dashboard";
  });

  it("renders icon always", () => {
    render(
      <SidebarNavItem href="/dashboard" icon="■" label="Dashboard" isExpanded={false} />
    );
    expect(screen.getByText("■")).toBeInTheDocument();
  });

  it("renders label when expanded", () => {
    render(
      <SidebarNavItem href="/dashboard" icon="■" label="Dashboard" isExpanded={true} />
    );
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("hides label when collapsed", () => {
    render(
      <SidebarNavItem href="/dashboard" icon="■" label="Dashboard" isExpanded={false} />
    );
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("shows active style when pathname matches", () => {
    mockPathname = "/dashboard";
    render(
      <SidebarNavItem href="/dashboard" icon="■" label="Dashboard" isExpanded={true} />
    );
    const link = screen.getByRole("link");
    expect(link.className).toContain("text-nexus-accent");
  });

  it("shows inactive style when pathname differs", () => {
    mockPathname = "/search";
    render(
      <SidebarNavItem href="/dashboard" icon="■" label="Dashboard" isExpanded={true} />
    );
    const link = screen.getByRole("link");
    expect(link.className).toContain("text-nexus-text-muted");
  });

  it("renders badge when badge prop is true", () => {
    render(
      <SidebarNavItem href="/alerts" icon="⚠" label="Alerts" isExpanded={true} badge />
    );
    expect(screen.getByTestId("nav-badge")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx jest src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx --verbose`
Expected: FAIL — module not found

- [ ] **Step 3: Implement SidebarNavItem**

```tsx
// src/widgets/app-sidebar/ui/SidebarNavItem.tsx
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

interface SidebarNavItemProps {
  href: string;
  icon: string;
  label: string;
  isExpanded: boolean;
  badge?: boolean;
}

export function SidebarNavItem({ href, icon, label, isExpanded, badge }: SidebarNavItemProps) {
  const pathname = usePathname();
  const isActive = pathname === href || pathname.startsWith(href + "/");

  return (
    <Link
      href={href}
      className={`relative flex items-center gap-3 rounded-lg h-10 transition-colors ${
        isExpanded ? "px-3" : "justify-center"
      } ${
        isActive
          ? "bg-nexus-sidebar-active text-nexus-accent"
          : "text-nexus-text-muted hover:bg-nexus-sidebar-hover hover:text-nexus-text-secondary"
      }`}
    >
      {isActive && (
        <span className="absolute left-0 w-[3px] h-5 bg-nexus-accent rounded-r" />
      )}
      <span className="text-lg flex-shrink-0 w-5 text-center">{icon}</span>
      {isExpanded && (
        <span className="text-sm font-medium whitespace-nowrap overflow-hidden">
          {label}
        </span>
      )}
      {badge && (
        <span
          data-testid="nav-badge"
          className="absolute top-1.5 right-1.5 w-2 h-2 bg-nexus-failure rounded-full border-2 border-nexus-sidebar"
        />
      )}
    </Link>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npx jest src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx --verbose`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add src/widgets/app-sidebar/ui/SidebarNavItem.tsx src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx
git commit -m "feat: add SidebarNavItem component with active state and badge"
```

---

### Task 4: AppSidebar + Sub-components (TDD)

**Files:**
- Create: `src/widgets/app-sidebar/ui/SidebarLogo.tsx`
- Create: `src/widgets/app-sidebar/ui/SidebarUserInfo.tsx`
- Create: `src/widgets/app-sidebar/ui/AppSidebar.tsx`
- Create: `src/widgets/app-sidebar/ui/SidebarLayout.tsx`
- Create: `src/widgets/app-sidebar/index.ts`
- Modify: `src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx`

- [ ] **Step 1: Add integration tests to existing test file**

`src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx` 하단에 추가:

```tsx
import { useSidebarStore } from "@/shared/model/sidebar.store";
import { AppSidebar } from "../ui/AppSidebar";
import userEvent from "@testing-library/user-event";

describe("AppSidebar", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarStore.setState(useSidebarStore.getInitialState());
  });

  it("renders all 9 navigation items", () => {
    render(<AppSidebar />);
    const links = screen.getAllByRole("link");
    // 9 nav items + logo link = 10
    expect(links.length).toBeGreaterThanOrEqual(9);
  });

  it("renders logo", () => {
    render(<AppSidebar />);
    expect(screen.getByTestId("sidebar-logo")).toBeInTheDocument();
  });

  it("expands on mouse enter", async () => {
    render(<AppSidebar />);
    const sidebar = screen.getByTestId("sidebar-nav");
    await userEvent.hover(sidebar);
    expect(useSidebarStore.getState().isExpanded).toBe(true);
  });

  it("collapses on mouse leave when not pinned", async () => {
    render(<AppSidebar />);
    const sidebar = screen.getByTestId("sidebar-nav");
    await userEvent.hover(sidebar);
    await userEvent.unhover(sidebar);
    expect(useSidebarStore.getState().isExpanded).toBe(false);
  });

  it("shows pin button when expanded", async () => {
    useSidebarStore.setState({ isExpanded: true });
    render(<AppSidebar />);
    expect(screen.getByTestId("pin-toggle")).toBeInTheDocument();
  });

  it("toggles pin on pin button click", async () => {
    useSidebarStore.setState({ isExpanded: true });
    render(<AppSidebar />);
    await userEvent.click(screen.getByTestId("pin-toggle"));
    expect(useSidebarStore.getState().isPinned).toBe(true);
  });
});
```

- [ ] **Step 2: Run tests to verify new tests fail**

Run: `npx jest src/widgets/app-sidebar/__tests__/AppSidebar.test.tsx --verbose`
Expected: FAIL — AppSidebar module not found

- [ ] **Step 3: Create SidebarLogo**

```tsx
// src/widgets/app-sidebar/ui/SidebarLogo.tsx
import Link from "next/link";

interface SidebarLogoProps {
  isExpanded: boolean;
}

export function SidebarLogo({ isExpanded }: SidebarLogoProps) {
  return (
    <Link href="/dashboard" data-testid="sidebar-logo" className="flex items-center gap-3 px-3 mb-4">
      <div className="w-9 h-9 rounded-[10px] bg-gradient-to-br from-nexus-accent to-purple-400 flex items-center justify-center font-extrabold text-sm text-white flex-shrink-0 shadow-[0_0_20px_rgba(99,102,241,0.25)]">
        N
      </div>
      {isExpanded && (
        <span className="text-sm font-bold text-nexus-text-primary whitespace-nowrap">
          NEXUS
        </span>
      )}
    </Link>
  );
}
```

- [ ] **Step 4: Create SidebarUserInfo**

```tsx
// src/widgets/app-sidebar/ui/SidebarUserInfo.tsx
interface SidebarUserInfoProps {
  isExpanded: boolean;
}

export function SidebarUserInfo({ isExpanded }: SidebarUserInfoProps) {
  return (
    <div className={`flex items-center gap-3 ${isExpanded ? "px-3" : "justify-center"}`}>
      <div className="w-7 h-7 rounded-full bg-gradient-to-br from-nexus-accent to-blue-400 flex items-center justify-center text-[11px] font-semibold text-white flex-shrink-0">
        U
      </div>
      {isExpanded && (
        <span className="text-xs text-nexus-text-secondary whitespace-nowrap">User</span>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Create AppSidebar**

```tsx
// src/widgets/app-sidebar/ui/AppSidebar.tsx
"use client";

import { useSidebarStore } from "@/shared/model/sidebar.store";
import { SidebarLogo } from "./SidebarLogo";
import { SidebarNavItem } from "./SidebarNavItem";
import { SidebarUserInfo } from "./SidebarUserInfo";

const NAV_ITEMS_TOP = [
  { href: "/dashboard", icon: "■", label: "Dashboard" },
  { href: "/search", icon: "◬", label: "Search" },
  { href: "/chart", icon: "╱", label: "Chart" },
  { href: "/pipeline", icon: "⚙", label: "Pipeline" },
  { href: "/cases", icon: "☰", label: "Cases" },
  { href: "/backtest", icon: "◔", label: "Backtest" },
  { href: "/portfolio", icon: "◆", label: "Portfolio" },
];

const NAV_ITEMS_BOTTOM = [
  { href: "/alerts", icon: "⚠", label: "Alerts", badge: true },
  { href: "/marketplace", icon: "★", label: "Marketplace" },
];

export function AppSidebar() {
  const { isPinned, isExpanded, togglePin, setExpanded } = useSidebarStore();
  const expanded = isPinned || isExpanded;

  return (
    <nav
      data-testid="sidebar-nav"
      className={`absolute inset-y-0 left-0 z-10 flex flex-col py-3 bg-nexus-sidebar border-r border-nexus-border transition-[width] duration-200 ${
        expanded ? "w-[200px]" : "w-16"
      } ${!isPinned && expanded ? "shadow-[4px_0_24px_rgba(0,0,0,0.5)]" : ""}`}
      onMouseEnter={() => setExpanded(true)}
      onMouseLeave={() => setExpanded(false)}
    >
      <SidebarLogo isExpanded={expanded} />

      <div className="flex flex-col gap-0.5 px-2">
        {NAV_ITEMS_TOP.map((item) => (
          <SidebarNavItem key={item.href} {...item} isExpanded={expanded} />
        ))}
      </div>

      <div className="flex-1" />

      <div className="flex flex-col gap-0.5 px-2 mb-2">
        {NAV_ITEMS_BOTTOM.map((item) => (
          <SidebarNavItem key={item.href} {...item} isExpanded={expanded} />
        ))}
      </div>

      <div className="border-t border-nexus-border pt-3 px-2">
        <SidebarUserInfo isExpanded={expanded} />
      </div>

      {expanded && (
        <button
          data-testid="pin-toggle"
          onClick={togglePin}
          className={`absolute top-3 right-2 w-6 h-6 rounded flex items-center justify-center text-xs transition-colors ${
            isPinned
              ? "text-nexus-accent bg-nexus-sidebar-active"
              : "text-nexus-text-muted hover:text-nexus-text-secondary"
          }`}
        >
          {isPinned ? "◉" : "○"}
        </button>
      )}
    </nav>
  );
}
```

- [ ] **Step 6: Create SidebarLayout**

```tsx
// src/widgets/app-sidebar/ui/SidebarLayout.tsx
"use client";

import { useSidebarStore } from "@/shared/model/sidebar.store";
import { AppSidebar } from "./AppSidebar";

export function SidebarLayout({ children }: { children: React.ReactNode }) {
  const isPinned = useSidebarStore((s) => s.isPinned);

  return (
    <div className="flex h-screen">
      <div
        className={`relative flex-shrink-0 transition-[width] duration-200 ${
          isPinned ? "w-[200px]" : "w-16"
        }`}
      >
        <AppSidebar />
      </div>
      <main className="flex-1 overflow-y-auto min-w-0 bg-nexus-bg">
        {children}
      </main>
    </div>
  );
}
```

- [ ] **Step 7: Create barrel export**

```typescript
// src/widgets/app-sidebar/index.ts
export { SidebarLayout } from "./ui/SidebarLayout";
export { AppSidebar } from "./ui/AppSidebar";
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `npx jest src/widgets/app-sidebar/__tests__/ --verbose`
Expected: All tests PASS (store + NavItem + AppSidebar)

- [ ] **Step 9: Commit**

```bash
git add src/widgets/app-sidebar/
git commit -m "feat: add AppSidebar widget with collapsible pin mode"
```

---

### Task 5: Route Group + Layout

**Files:**
- Create: `src/app/(app)/layout.tsx`
- Modify: `src/app/layout.tsx`

@AGENTS.md — Next.js 16 레이아웃 관련 문서를 먼저 확인할 것: `node_modules/next/dist/docs/`

- [ ] **Step 1: Create (app) route group layout**

```tsx
// src/app/(app)/layout.tsx
import { SidebarLayout } from "@/widgets/app-sidebar";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <SidebarLayout>{children}</SidebarLayout>;
}
```

- [ ] **Step 2: Simplify root layout**

`src/app/layout.tsx` — `<div className="flex flex-col min-h-screen">` wrapper를 제거:

변경 전:
```tsx
<body className="font-sans bg-nexus-bg text-nexus-text-primary min-h-screen antialiased">
  <div className="flex flex-col min-h-screen">
    {children}
  </div>
</body>
```

변경 후:
```tsx
<body className="font-sans bg-nexus-bg text-nexus-text-primary min-h-screen antialiased">
  {children}
</body>
```

- [ ] **Step 3: Commit**

```bash
git add src/app/\(app\)/layout.tsx src/app/layout.tsx
git commit -m "feat: add (app) route group with sidebar layout"
```

---

### Task 6: Page Migration

**Files:**
- Move: `src/app/(pages)/search/page.tsx` → `src/app/(app)/search/page.tsx`
- Move: `src/app/(pages)/chart/*` → `src/app/(app)/chart/*`
- Move: `src/app/(pages)/cases/*` → `src/app/(app)/cases/*`
- Move: `src/app/pipeline/*` → `src/app/(app)/pipeline/*`
- Create 5 placeholder pages
- Modify: `src/app/page.tsx` — redirect

- [ ] **Step 1: Create (app) directory structure**

```bash
mkdir -p src/app/\(app\)/search
mkdir -p src/app/\(app\)/chart
mkdir -p src/app/\(app\)/pipeline
mkdir -p src/app/\(app\)/cases/\[id\]
mkdir -p src/app/\(app\)/dashboard
mkdir -p src/app/\(app\)/backtest
mkdir -p src/app/\(app\)/portfolio
mkdir -p src/app/\(app\)/alerts
mkdir -p src/app/\(app\)/marketplace
```

- [ ] **Step 2: Move existing pages**

```bash
# Search
mv src/app/\(pages\)/search/page.tsx src/app/\(app\)/search/page.tsx

# Chart (layout + page)
mv src/app/\(pages\)/chart/layout.tsx src/app/\(app\)/chart/layout.tsx
mv src/app/\(pages\)/chart/page.tsx src/app/\(app\)/chart/page.tsx

# Cases (page + [id])
mv src/app/\(pages\)/cases/page.tsx src/app/\(app\)/cases/page.tsx
mv src/app/\(pages\)/cases/\[id\]/page.tsx src/app/\(app\)/cases/\[id\]/page.tsx

# Pipeline (layout + page)
mv src/app/pipeline/layout.tsx src/app/\(app\)/pipeline/layout.tsx
mv src/app/pipeline/page.tsx src/app/\(app\)/pipeline/page.tsx
```

- [ ] **Step 3: Create placeholder pages**

Dashboard:
```tsx
// src/app/(app)/dashboard/page.tsx
export default function DashboardPage() {
  return (
    <div className="flex flex-1 items-center justify-center h-full">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

Backtest:
```tsx
// src/app/(app)/backtest/page.tsx
export default function BacktestPage() {
  return (
    <div className="flex flex-1 items-center justify-center h-full">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Backtest</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

Portfolio:
```tsx
// src/app/(app)/portfolio/page.tsx
export default function PortfolioPage() {
  return (
    <div className="flex flex-1 items-center justify-center h-full">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Portfolio</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

Alerts:
```tsx
// src/app/(app)/alerts/page.tsx
export default function AlertsPage() {
  return (
    <div className="flex flex-1 items-center justify-center h-full">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Alerts</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

Marketplace:
```tsx
// src/app/(app)/marketplace/page.tsx
export default function MarketplacePage() {
  return (
    <div className="flex flex-1 items-center justify-center h-full">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Marketplace</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Update root page to redirect**

@AGENTS.md — `redirect()` 사용 전 `node_modules/next/dist/docs/`에서 redirect 문서 확인할 것.

```tsx
// src/app/page.tsx
import { redirect } from "next/navigation";

export default function Home() {
  redirect("/dashboard");
}
```

- [ ] **Step 5: Verify dev server starts without errors**

Run: `npx next build` (또는 `npm run dev`로 확인)
Expected: 빌드 성공, 콘솔 에러 없음

- [ ] **Step 6: Commit**

```bash
git add src/app/
git commit -m "feat: migrate pages to (app) route group with sidebar layout"
```

---

### Task 7: Cleanup

**Files:**
- Delete: `src/app/(pages)/` (empty directory)
- Delete: `src/app/pipeline/` (empty directory)
- Delete: `src/shared/ui/AppNavigation.tsx`

- [ ] **Step 1: Remove old directories and files**

```bash
rm -rf src/app/\(pages\)
rm -rf src/app/pipeline
rm src/shared/ui/AppNavigation.tsx
```

- [ ] **Step 2: Verify no broken imports or stale references**

`AppNavigation` 및 `(pages)` 참조가 남아있지 않음을 확인:

Run: `grep -r "AppNavigation" src/ --include="*.tsx" --include="*.ts"`
Expected: 결과 없음

Run: `grep -r "(pages)" src/ --include="*.tsx" --include="*.ts"`
Expected: 결과 없음

- [ ] **Step 3: Run all unit tests**

Run: `npx jest --verbose`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove old route groups and unused AppNavigation"
```

---

### Task 8: E2E Tests

**Files:**
- Modify: `e2e/landing.spec.ts` — 루트가 redirect하므로 업데이트
- Create: `e2e/sidebar-navigation.spec.ts`

- [ ] **Step 1: Update landing test**

`e2e/landing.spec.ts` — `/`가 `/dashboard`로 redirect되므로, 기존 8개 테스트를 교체한다.
기존 landing 페이지("NEXUS" heading, "System Online" indicator, pulse animation 등)는 삭제되므로 해당 테스트도 함께 제거.
새 테스트는 redirect 동작과 기본 HTML 속성만 검증:

```typescript
// e2e/landing.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Landing Page Redirect", () => {
  test("redirects / to /dashboard", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test("has Korean language attribute", async ({ page }) => {
    await page.goto("/");
    const html = page.locator("html");
    await expect(html).toHaveAttribute("lang", "ko");
  });

  test("has dark class on html element", async ({ page }) => {
    await page.goto("/");
    const html = page.locator("html");
    await expect(html).toHaveClass(/dark/);
  });

  test("has correct page title", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/NEXUS/);
  });
});
```

- [ ] **Step 2: Create sidebar navigation E2E test**

```typescript
// e2e/sidebar-navigation.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Sidebar Navigation", () => {
  test("sidebar is visible on all pages", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");
    await expect(sidebar).toBeVisible();
  });

  test("sidebar shows logo", async ({ page }) => {
    await page.goto("/dashboard");
    const logo = page.getByTestId("sidebar-logo");
    await expect(logo).toBeVisible();
  });

  test("clicking nav item navigates to page", async ({ page }) => {
    await page.goto("/dashboard");
    const searchLink = page.getByRole("link", { name: /search/i });
    await searchLink.click();
    await expect(page).toHaveURL(/\/search/);
  });

  test("active nav item is highlighted", async ({ page }) => {
    await page.goto("/dashboard");
    const dashboardLink = page.getByRole("link", { name: /dashboard/i });
    await expect(dashboardLink).toHaveClass(/text-nexus-accent/);
  });

  test("sidebar expands on hover", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");

    // Check initial collapsed width
    const initialWidth = await sidebar.evaluate((el) => el.offsetWidth);
    expect(initialWidth).toBe(64);

    // Hover to expand
    await sidebar.hover();
    await page.waitForTimeout(300); // wait for transition

    const expandedWidth = await sidebar.evaluate((el) => el.offsetWidth);
    expect(expandedWidth).toBe(200);
  });

  test("pin toggle keeps sidebar expanded", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");

    // Hover to show pin button
    await sidebar.hover();
    await page.waitForTimeout(300);

    // Click pin
    const pinBtn = page.getByTestId("pin-toggle");
    await pinBtn.click();

    // Move mouse away
    await page.mouse.move(500, 500);
    await page.waitForTimeout(300);

    // Should still be expanded
    const width = await sidebar.evaluate((el) => el.offsetWidth);
    expect(width).toBe(200);
  });

  test("placeholder pages show coming soon", async ({ page }) => {
    await page.goto("/backtest");
    await expect(page.locator("text=Coming soon")).toBeVisible();
  });
});
```

- [ ] **Step 3: Run E2E tests**

Run: `npx playwright test e2e/landing.spec.ts e2e/sidebar-navigation.spec.ts`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add e2e/landing.spec.ts e2e/sidebar-navigation.spec.ts
git commit -m "test: update landing tests and add sidebar navigation e2e tests"
```
