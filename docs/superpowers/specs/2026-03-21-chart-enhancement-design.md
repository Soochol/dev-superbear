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
- 좌측: 텍스트 사이드메뉴 (종목 검색 / 관심 종목 / 최근 본 종목)
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

**검색 데이터 소스:**
- `stocks` 테이블의 전체 종목 목록을 백엔드에서 제공하는 API 필요
- `GET /api/v1/stocks/search?q=삼성` — 종목명/코드 기반 검색 (trigram 인덱스 활용)
- 프론트에서는 검색어 입력 시 300ms debounce 후 API 호출
- 빈 검색어 시 인기 종목 또는 빈 리스트 표시

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
- `fetchWatchlist()` → `GET /api/v1/watchlist` → `WatchlistItem[]`
- `addWatchlistItem(symbol, name)` → `POST /api/v1/watchlist`
- `removeWatchlistItem(symbol)` → `DELETE /api/v1/watchlist/:symbol`

**타입 변환:**
백엔드 `WatchlistItem`(`{id, user_id, symbol, name, created_at}`)은 프론트 `SearchResult`와 다르다.
프론트에서 변환 레이어를 둔다:
```typescript
function toSearchResult(item: WatchlistItem): SearchResult {
  return {
    symbol: item.symbol,
    name: item.name,
    matchedValue: item.symbol,  // 필수 필드, symbol로 채움
  };
}
```

**Store 변경** (`useStockListStore`):
- `loadWatchlist()` 액션 추가 — API fetch → `toSearchResult()` 변환 → `watchlist` 상태 업데이트
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

**백엔드 변경 — KIS API 엔드포인트 분기:**
현재 KIS 클라이언트는 일봉 전용 엔드포인트(`inquire-daily-itemchartprice`, TR: `FHKST03010100`)만 사용한다.
분봉 데이터는 별도 엔드포인트가 필요하다:
- 분봉: `inquire-time-itemchartprice` (TR: `FHKST03010200`), `FID_INPUT_HOUR_1` 파라미터
- 일/주/월봉: 기존 `inquire-daily-itemchartprice` (TR: `FHKST03010100`), `FID_PERIOD_DIV_CODE` 파라미터

`CandleService`에 타임프레임 기반 라우팅 레이어 추가:
```
timeframe → isIntraday(tf)?
  → yes: KIS 분봉 API (1m, 5m, 15m, 30m)
  → no:  KIS 일봉 API (1D, 1W, 1M)
```

4H 캔들은 KIS에 네이티브 지원이 없으므로 1H 캔들 4개를 집계하여 합성한다.
프론트 → 백엔드 타임프레임 매핑:
| Frontend | KIS API | 비고 |
|----------|---------|------|
| 1m | 분봉 API, interval=1 | |
| 5m | 분봉 API, interval=5 | |
| 15m | 분봉 API, interval=15 | |
| 30m | 분봉 API, interval=30 | |
| 1H | 분봉 API, interval=60 | |
| 4H | 분봉 API, interval=60 → 4개 집계 | 서버사이드 합성 |
| 1D | 일봉 API, period=D | |
| 1W | 일봉 API, period=W | |
| 1M | 일봉 API, period=M | |

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
- Bollinger Bands 추가: upper/middle/lower 3개 라인 (lightweight-charts에서 area fill 미지원이므로 라인만 렌더링)
- `overlayConfigs`에 Bollinger Bands 설정 추가

**레이아웃 — 수직 스택:**
```
┌─────────────────────────────┐
│ ChartTopbar (종목 + TF + 지표)  │  고정 높이
├─────────────────────────────┤
│                             │
│ MainChart + Overlays        │  flex: 1 (남은 공간 차지)
│                             │
├── PanelResizer ─────────────┤  드래그 가능 (8px)
│ RSI Panel (헤더 + 차트)      │  기본 120px, min 80px
├── PanelResizer ─────────────┤
│ MACD Panel (헤더 + 차트)     │  기본 120px, min 80px
├─────────────────────────────┤
│ BottomInfoPanel             │  고정 높이 (기존)
└─────────────────────────────┘
```
- 패널이 0개일 때: MainChart가 전체 높이
- 패널이 1-2개일 때: MainChart가 축소되며 패널이 하단에 스택
- `PanelResizer`로 각 패널 높이 조절 가능

**Panel 렌더링 (새 컴포넌트):**
- `lightweight-charts`의 별도 차트 인스턴스로 RSI/MACD 렌더링
- 메인 차트와 시간축 동기화: `timeScale().subscribeVisibleLogicalRangeChange` + `subscribeCrosshairMove` (양방향)
- 각 패널: 헤더(지표명 + 닫기 버튼) + 차트 영역

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

**지표 레지스트리** (`entities/indicator/model/registry.ts`):
```typescript
interface IndicatorConfig {
  id: string;             // "ma5", "ma20", "rsi", "macd", "bb"
  name: string;           // "MA(5)", "RSI(14)", "MACD(12,26,9)", "BB(20,2)"
  category: "moving-average" | "oscillator" | "band";
  type: "overlay" | "panel";
  defaultParams: Record<string, number>;
  calculate: (closes: number[], params: Record<string, number>) => IndicatorResult;
}
```
- 위치: `entities/indicator/` (FSD entities 레이어 — 도메인 모델)
- `activeIndicators` store의 ID와 레지스트리 ID 동일 체계 사용 (예: `"rsi"`, `"macd"`, `"bb"`, `"ma20"`)
- 1차에서는 파라미터 고정 (기본값 사용), 사용자 설정은 향후 확장

**1차 지원 지표:**
- MA (5/20/60/120/200) — overlay ✅ 기존 동작
- Bollinger Bands (20, 2) — overlay 🆕 (3개 라인)
- RSI (14) — panel 🆕
- MACD (12, 26, 9) — panel 🆕

---

## 구현 순서

1. **종목 검색 팝업** — 새 UI + 기존 사이드바 제거
2. **관심종목 DB 연동** — API 클라이언트 + store 연동
3. **타임프레임 완성** — UI 개선 + 30m/4H 추가 + 백엔드 분봉 API 연동
4. **보조지표 시스템** — 렌더링 완성 + 선택 UI + 패널

서브시스템 1→2는 의존성 있음 (팝업 안에 관심종목 UI). 3, 4는 독립적.

## 테스트 전략

**기존 테스트 마이그레이션:**
- `StockListSidebar.test.tsx` — 삭제 (모달로 대체)
- `ChartPageLayout.test.tsx` — 사이드바 mock 제거, 새 레이아웃에 맞게 업데이트

**새 테스트:**
- 서브시스템 1: `StockSearchModal.test.tsx` — 모달 열기/닫기, 탭 전환, 종목 선택 시 차트 연동
- 서브시스템 2: `watchlist-api.test.ts` — API 클라이언트 호출/에러 처리, store 연동
- 서브시스템 3: `ChartTopbar.test.tsx` — 타임프레임 버튼 그룹 렌더링, 선택 동작
- 서브시스템 4: `IndicatorSelector.test.tsx` — 지표 선택/해제, `IndicatorPanel.test.tsx` — 패널 렌더링

**E2E 테스트:**
- 종목 검색 → 차트 전환 플로우
- 관심종목 추가/삭제 → 새로고침 후 유지 확인
- 타임프레임 변경 → 차트 데이터 갱신 확인

## 기술 스택

- **차트 라이브러리:** lightweight-charts v5.1.0 (기존)
- **상태 관리:** Zustand v5.0.12 (기존)
- **스타일:** Tailwind CSS 4 + nexus 디자인 토큰 (기존)
- **FSD 구조:** entities → features → widgets 레이어 (기존)
- **백엔드:** Go/Gin, KIS API, PostgreSQL (기존)
