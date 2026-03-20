# 차트 기능 강화 디자인 스펙

> 날짜: 2026-03-21
> 상태: Approved
> 레퍼런스: TradingView UIUX

## 개요

nexus 플랫폼의 차트 기능을 TradingView 수준으로 강화한다. 4개 독립 서브시스템을 순차적으로 구현하며, 각 단계가 독립적으로 동작 가능하다.

## 접근법

순차적 기능 완성 (접근법 A). 4개 서브시스템을 독립 구현하되, 하나의 스펙에서 관리한다.

---

## 서브시스템 1: 종목 검색 팝업

### 문제

현재 차트 우측에 고정된 `StockListSidebar`(w-72)가 차트 영역을 차지하며, TradingView 스타일의 센터 팝업 UX와 거리가 있다.

### 설계

**레이아웃 변경:**
- `ChartPageLayout`에서 `StockListSidebar` 제거
- 차트가 전체 너비를 사용
- 종목 검색은 센터 모달로 전환

**모달 구조 — 텍스트 사이드바 스타일:**
- 화면 중앙에 위치하는 모달 (backdrop 포함)
- 좌측: 텍스트 사이드메뉴 (종목 검색 / 관심 종목 / 최근 본 종목 / 설정)
- 우측: 제목 + 검색 인풋 + 종목 리스트 + 하단 키보드 힌트
- 종목 아이템: 종목명, 코드, 현재가, 등락률, 관심종목 star 토글

**모달 트리거:**
- `ChartTopbar` 좌측 종목 영역 클릭
- 종목 미선택 시: `"종목을 검색하세요"` placeholder + 검색 아이콘
- 종목 선택 후: `symbol | name | price | change%` 표시, 클릭 시 모달 열림

**모달 동작:**
- 열기: ChartTopbar 종목 영역 클릭
- 닫기: Esc, backdrop 클릭, 종목 선택 시 자동 닫힘
- 키보드: ↑↓ 이동, Enter 선택, Esc 닫기
- 종목 클릭 → `setCurrentStock()` + `addToRecent()` → 모달 닫힘 → 차트 전환

**컴포넌트 구조:**
```
widgets/stock-search-modal/
├── ui/
│   ├── StockSearchModal.tsx      # 모달 컨테이너 (backdrop + 센터 정렬)
│   ├── SearchSideNav.tsx         # 좌측 텍스트 사이드메뉴
│   ├── SearchContent.tsx         # 우측 컨텐츠 (입력 + 리스트)
│   ├── SearchInput.tsx           # 검색 인풋
│   └── SearchStockItem.tsx       # 종목 아이템 행
├── model/
│   └── search-modal.store.ts     # 모달 열기/닫기 + 활성 탭 상태
└── index.ts
```

**상태 관리:**
- 새 store: `search-modal.store.ts` — `isOpen`, `activeTab`, `openModal()`, `closeModal()`
- 기존 `useStockListStore`의 `searchResults`, `watchlist`, `recentStocks` 재사용
- 기존 `useChartStore`의 `setCurrentStock()` 재사용

**삭제 대상:**
- `widgets/stock-list-sidebar/` — 전체 디렉토리 (기능이 모달로 이전)
- `ChartPageLayout.tsx`에서 사이드바 관련 코드 제거

---

## 서브시스템 2: 관심종목 DB 영속화

### 문제

현재 관심종목이 Zustand 인메모리에만 저장되어 새로고침 시 사라진다. 백엔드 API(`WatchlistHandler`)와 DB 스키마(`009_watchlist.sql`)는 이미 준비되어 있다.

### 설계

**API 클라이언트** (`features/watchlist/api/watchlist-api.ts`):
- `fetchWatchlist()` → `GET /api/v1/watchlist` → `SearchResult[]`
- `addWatchlistItem(symbol, name)` → `POST /api/v1/watchlist`
- `removeWatchlistItem(symbol)` → `DELETE /api/v1/watchlist/:symbol`

**Store 변경** (`useStockListStore`):
- `loadWatchlist()` 액션 추가 — API fetch 후 `watchlist` 상태 업데이트
- `addToWatchlist()` — API 호출 성공 후 로컬 상태 업데이트
- `removeFromWatchlist()` — API 호출 성공 후 로컬 상태 업데이트
- 기존 인터페이스(`isInWatchlist` 등) 유지

**초기 로드:**
- 검색 모달이 처음 열릴 때 `loadWatchlist()` 호출 (lazy load)
- 이후 캐싱된 데이터 사용, 추가/삭제 시 로컬 즉시 반영

**데이터 흐름:**
```
모달 열림 → 관심종목 탭 → 캐싱된 watchlist 표시
☆ 클릭 → POST /watchlist → 성공 → store 업데이트 → UI 반영
★ 클릭 → DELETE /watchlist/:symbol → 성공 → store 업데이트 → UI 반영
API 실패 → 에러 무시, 로컬 상태 변경 없음 (서버 확인 후 반영 원칙)
```

---

## 서브시스템 3: 타임프레임 완성

### 문제

데이터 파이프라인(`useChartData` → `chartApi.fetchCandles` → KIS API)은 이미 연결되어 있으나, UI가 TradingView 스타일과 거리가 있고, `30m`/`4H` 타임프레임이 없다.

### 설계

**타임프레임 목록 확장:**
- 현재: `1m`, `5m`, `15m`, `1H`, `1D`, `1W`, `1M` (7개)
- 변경: `1m`, `5m`, `15m`, `30m`, `1H`, `4H`, `1D`, `1W`, `1M` (9개)

**UI 개선 — TradingView 스타일 버튼 그룹:**
- 분봉 그룹: `1` `5` `15` `30` (라벨: 숫자만, 단위는 그룹으로 구분)
- 시봉 그룹: `1H` `4H`
- 일/주/월: `D` `W` `M`
- 구분선으로 그룹 시각 분리
- 현재 선택 타임프레임: accent 색상 하이라이트

**백엔드 변경:**
- KIS API period 매핑에 `30m`, `4H` 추가
- 나머지 데이터 흐름은 기존과 동일

**Timeframe 타입 확장:**
```typescript
type Timeframe = "1m" | "5m" | "15m" | "30m" | "1H" | "4H" | "1D" | "1W" | "1M";
```

---

## 서브시스템 4: 보조지표 시스템

### 문제

MA만 차트에 오버레이로 렌더링됨. RSI, MACD, Bollinger Bands는 계산 로직만 있고 차트에 그려지지 않음. 지표 선택 UI가 없어 하드코딩된 `["ma20", "ma60"]`만 활성화됨.

### 설계

**지표 분류:**

| 유형 | 지표 | 렌더링 위치 |
|------|------|------------|
| Overlay | MA(5/20/60/120/200), Bollinger Bands | 메인 캔들 차트 위에 겹침 |
| Panel | RSI, MACD | 차트 아래 별도 패널 |

**지표 선택 UI:**
- `ChartTopbar`에 "지표" 버튼 추가
- 클릭 시 지표 목록 팝오버 표시
- 카테고리별 그룹: 이동평균 / 오실레이터 / 밴드
- 체크박스로 활성/비활성 토글
- 활성 지표: 차트 상단 라벨로 표시, 클릭 시 제거

**Overlay 렌더링 (MainChart.tsx 확장):**
- 기존 MA 오버레이 로직 유지
- Bollinger Bands 추가: upper/middle/lower 3개 라인 + 영역 fill
- `overlayConfigs`에 Bollinger Bands 설정 추가

**Panel 렌더링 (새 컴포넌트):**
- `lightweight-charts`의 별도 차트 인스턴스로 RSI/MACD 렌더링
- 메인 차트와 시간축 동기화 (`timeScale().subscribeVisibleLogicalRangeChange`)
- 각 패널: 헤더(지표명 + 닫기 버튼) + 차트 영역
- 드래그로 패널 높이 조절 (`PanelResizer`)

**RSI 패널:**
- Line series (0-100 범위)
- 과매수(70)/과매도(30) 수평선
- 기본 period: 14

**MACD 패널:**
- MACD line (blue) + Signal line (orange) + Histogram (bar series, green/red)
- 기본 periods: 12, 26, 9

**컴포넌트 구조:**
```
features/chart/ui/
├── MainChart.tsx              # overlay 지표 렌더링 강화 (BB 추가)
├── IndicatorPanel.tsx         # RSI/MACD 별도 패널 컨테이너
├── IndicatorPanelHeader.tsx   # 패널 헤더 (이름 + 닫기)
└── PanelResizer.tsx           # 드래그 리사이저

widgets/main-chart/ui/
├── ChartTopbar.tsx            # 지표 버튼 추가
├── IndicatorSelector.tsx      # 지표 선택 팝오버
└── ChartPageLayout.tsx        # 레이아웃에 패널 영역 추가
```

**지표 레지스트리:**
```typescript
interface IndicatorConfig {
  id: string;
  name: string;
  category: "moving-average" | "oscillator" | "band";
  type: "overlay" | "panel";
  defaultParams: Record<string, number>;
  calculate: (closes: number[], params: Record<string, number>) => IndicatorResult;
}
```

각 지표를 레지스트리에 등록하면, 선택 UI와 렌더링이 자동으로 연동된다. 향후 새 지표 추가 시 레지스트리에 등록만 하면 된다.

**1차 지원 지표:**
- MA (5/20/60/120/200) — overlay ✅ 기존 동작
- Bollinger Bands (20, 2) — overlay 🆕
- RSI (14) — panel 🆕
- MACD (12, 26, 9) — panel 🆕

---

## 구현 순서

1. **종목 검색 팝업** — 새 UI + 기존 사이드바 제거
2. **관심종목 DB 연동** — API 클라이언트 + store 연동
3. **타임프레임 완성** — UI 개선 + 30m/4H 추가
4. **보조지표 시스템** — 렌더링 완성 + 선택 UI + 패널

서브시스템 1→2는 의존성 있음 (팝업 안에 관심종목 UI). 3, 4는 독립적.

## 기술 스택

- **차트 라이브러리:** lightweight-charts v5.1.0 (기존)
- **상태 관리:** Zustand v5.0.12 (기존)
- **스타일:** Tailwind CSS 4 + nexus 디자인 토큰 (기존)
- **FSD 구조:** entities → features → widgets 레이어 (기존)
- **백엔드:** Go/Gin, KIS API, PostgreSQL (기존)
