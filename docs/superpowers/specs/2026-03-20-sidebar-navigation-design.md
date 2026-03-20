# Sidebar Navigation Design

프로토타입(`prototype/index.html`)의 사이드 메뉴 레이아웃을 Next.js 프로젝트에 적용한다.

## 결정 사항

| 항목 | 결정 |
|------|------|
| 범위 | 프로토타입 9개 페이지 전체. 미구현 5개는 placeholder |
| 사이드바 스타일 | 접이식 64px ↔ 200px |
| 확장 방식 | 핀 모드 — hover 임시 확장 + 핀 버튼으로 고정 |
| Topbar | 제거. 검색/유저 정보를 사이드바에 통합 |
| 구현 접근법 | Route Group 분리 (`(app)/`) |
| 색상 체계 | 기존 nexus 테마(indigo 기반) 유지 |

## 라우팅 구조

```
src/app/
├── layout.tsx                ← 루트 (html, body, 폰트만)
├── page.tsx                  ← / → /dashboard redirect
├── (app)/                    ← 사이드바 포함 Route Group
│   ├── layout.tsx            ← AppSidebar + 메인 콘텐츠 영역
│   ├── dashboard/page.tsx         ← placeholder
│   ├── search/page.tsx            ← 기존 마이그레이션
│   ├── chart/
│   │   ├── layout.tsx
│   │   └── page.tsx               ← 기존 마이그레이션
│   ├── pipeline/
│   │   ├── layout.tsx
│   │   └── page.tsx               ← 기존 마이그레이션
│   ├── cases/
│   │   ├── page.tsx               ← 기존 마이그레이션
│   │   └── [id]/page.tsx
│   ├── backtest/page.tsx          ← placeholder
│   ├── portfolio/page.tsx         ← placeholder
│   ├── alerts/page.tsx            ← placeholder
│   └── marketplace/page.tsx       ← placeholder
```

기존 `(pages)/` route group과 `pipeline/` 독립 라우트를 `(app)/`으로 통합한다.

## 사이드바 컴포넌트 설계

### Zustand Store

```typescript
interface SidebarState {
  isPinned: boolean;      // 핀 고정 여부 (localStorage 영속)
  isExpanded: boolean;    // 현재 확장 상태
  togglePin: () => void;
  setExpanded: (v: boolean) => void;
}
```

- `isPinned: false` + hover → 임시 확장(overlay), 마우스 떠나면 축소
- `isPinned: true` → 항상 200px 확장, 콘텐츠 리플로우
- pinned 상태를 `localStorage`로 세션 간 유지

### 컴포넌트 트리 (FSD)

```
src/shared/model/
└── sidebar.store.ts              ← Zustand store (FSD: shared 레이어)

src/widgets/app-sidebar/
├── ui/
│   ├── AppSidebar.tsx            ← 메인 컨테이너 (64px ↔ 200px)
│   ├── SidebarLayout.tsx         ← 사이드바 + 콘텐츠 레이아웃
│   ├── SidebarLogo.tsx           ← N 로고
│   ├── SidebarNavItem.tsx        ← 개별 메뉴 아이템 (아이콘 + 라벨). Search도 NavItem으로 통합
│   └── SidebarUserInfo.tsx       ← 유저 (축소: 아바타, 확장: 이름 + 아바타)
```

### 메뉴 아이템

| 순서 | 라벨 | 경로 | 위치 |
|------|------|------|------|
| 1 | Dashboard | `/dashboard` | 상단 |
| 2 | Search | `/search` | 상단 |
| 3 | Chart | `/chart` | 상단 |
| 4 | Pipeline | `/pipeline` | 상단 |
| 5 | Cases | `/cases` | 상단 |
| 6 | Backtest | `/backtest` | 상단 |
| 7 | Portfolio | `/portfolio` | 상단 |
| — | spacer | — | — |
| 8 | Alerts | `/alerts` | 하단 (badge) |
| 9 | Marketplace | `/marketplace` | 하단 |

### 레이아웃 동작

- **축소 (64px)**: `grid-template-columns: 64px 1fr`
- **핀 확장 (200px)**: `grid-template-columns: 200px 1fr` — 콘텐츠 리플로우
- **hover 임시 확장**: 사이드바가 overlay로 200px, 콘텐츠는 64px 기준 유지

## 스타일링

기존 nexus 테마를 유지하고, 사이드바용 토큰만 추가한다.

```css
--color-nexus-sidebar: #0e0e16;
--color-nexus-sidebar-hover: #1a1a28;
--color-nexus-sidebar-active: rgba(99, 102, 241, 0.1);
```

- width 전환: `transition: width 200ms ease`
- 라벨 페이드: `transition: opacity 150ms ease`
- active 아이템: 좌측 3px accent bar + accent 배경

## 페이지 마이그레이션

| 페이지 | 현재 경로 | 변경 경로 | 작업 |
|--------|----------|----------|------|
| Search | `(pages)/search` | `(app)/search` | 디렉토리 이동 |
| Chart | `(pages)/chart` | `(app)/chart` | 디렉토리 이동 + layout 유지 |
| Pipeline | `pipeline/` | `(app)/pipeline` | route group 안으로 이동 |
| Cases | `(pages)/cases` | `(app)/cases` | 디렉토리 이동 + `[id]` 포함 |

- 내부 컴포넌트(widgets, features, entities)는 변경 없음
- `@/` alias 기반 import 경로 영향 없음
- 기존 `AppNavigation` 상단바 제거
- 마이그레이션 완료 후 빈 `(pages)/` 디렉토리 및 `pipeline/` 디렉토리 삭제

## Placeholder 페이지

Dashboard, Backtest, Portfolio, Alerts, Marketplace — 동일 패턴:

```tsx
export default function XxxPage() {
  return (
    <div className="flex flex-1 items-center justify-center">
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">Xxx</h1>
        <p className="text-nexus-text-muted text-sm">Coming soon</p>
      </div>
    </div>
  );
}
```

루트 `/`는 `/dashboard`로 redirect.

## 테스트

- `AppSidebar` 단위 테스트: 메뉴 렌더링, active 상태, 핀 토글
- 기존 E2E 테스트: 라우트 경로 업데이트
- Playwright: 사이드바 네비게이션 테스트 추가

## 스코프 밖

- 키보드 단축키 (Ctrl+K 등)
- 사이드바 드래그 리사이즈
- 모바일 반응형
