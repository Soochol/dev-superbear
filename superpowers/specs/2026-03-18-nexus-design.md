# NEXUS — AI-Native Investment Intelligence Platform

## Overview

기존 HTS/TradingView 패러다임을 완전히 탈피한 새로운 투자 분석 플랫폼.
에이전트 기반 파이프라인으로 종목 탐지 → 다차원 분석 → 장기 모니터링 → 백테스트까지 통합.

### Target Users

- 고급 트레이더 / 퀀트
- 투자 리서치 팀
- AI-네이티브 투자자 (에이전트에게 리서치를 위임)
- 전략 마켓플레이스 참여자

### Core Principle

- 모든 분석 블록은 **AI 에이전트**
- 에이전트는 **도구(tool)**를 자율적으로 선택하여 실행 (DSL, KIS API, DART, 뉴스 등)
- 사용자는 **자연어**로 에이전트 블록을 정의
- 파이프라인은 에이전트 블록의 **순차 + 병렬 오케스트레이션**

---

## Features

### 1. 종목 검색

에이전트 기반 조건 검색 시스템. **자연어/DSL 2탭 모드 공존형** UI.

#### 1.1 자연어 모드
- 사용자가 자연어로 검색 조건을 정의
  - 예: "최근 5년 안에 2년 최대거래량이 발생하고 거래대금 3000억 이상인 종목"
- 에이전트가 DSL 도구를 사용해 조건을 작성하고 실행
- 프리셋 칩 제공 (2yr Max Volume, Golden Cross, RSI Oversold 등)
- Agent 상태 표시 (Interpreting → Building → Scanning)

#### 1.2 DSL 모드
- 사용자가 직접 DSL 쿼리를 작성/편집
- 코드 에디터 스타일 (monospace, 다크 배경)
- 자동완성 힌트: 사용 가능한 함수/변수 (max_volume, trade_value, rsi, ma 등)
- [Validate] 버튼: 에이전트가 문법 검증
- [Explain in NL] 버튼: DSL → 자연어 설명

#### 1.3 LIVE DSL 패널
- 어떤 모드에서든 **항상 하단에 표시**
- 자연어 모드: 에이전트가 생성한 DSL을 구문 하이라이팅으로 표시
- DSL 모드: 사용자가 입력한 DSL이 실시간 반영
- 상태: "✓ Agent validated" / "⚠ Not validated"
- [Copy] + [Run Search] 버튼

#### 1.4 기타
- 검색 조건 저장/불러오기
- 마켓플레이스에서 다른 사용자의 검색 조건 사용 가능

### 2. 차트

검색된 종목을 시각적으로 확인하는 차트 시스템.

#### 2.1 레이아웃

```
┌──────────────────────────────────────────────┐
│ 005930  Samsung  78,400 +2.1%    1D 1W 1M    │  ← topbar
├──────────────────────────┬───────────────────┤
│                          │ [검색결과][★관심][최근]│  ← 종목 리스트 패널
│    MAIN CHART            │ 🔍 종목 검색...       │
│    (PER Band + MA)       │ ● Samsung  ← 활성   │
│                          │   ecoprobm  ★      │
├──────────────────────────┤   SK Hynix  ★      │
│ [RSI] [MACD] [Revenue]  │   ...              │
│  보조 지표 차트 영역      │                    │
├──────────────────────────┴───────────────────┤
│ FINANCIALS   │  AI FUSION    │ SECTOR COMPARE │  ← 하단 full-width 3칼럼
└──────────────────────────────────────────────┘
```

#### 2.2 종목 리스트 사이드바 (오른쪽)
- **3탭**: 검색결과 / ★ 관심종목 / 최근
- Search에서 "Chart" 클릭 → 차트 페이지 이동 + 검색결과 탭에 전체 결과 로드
- 종목 클릭 시 차트 즉시 전환
- ★ 아이콘으로 관심종목 추가/제거

#### 2.3 차트 영역 (왼쪽)
- 캔들스틱 차트 (일/주/월/분봉)
- 기술적 지표 오버레이 (MA, RSI, MACD, BB 등)
- **재무 이벤트 오버레이**
  - 실적 발표일, 유상증자, 배당기준일 등을 차트 위에 마커로 표시
  - PER/PBR 밴드 오버레이
- **보조 지표 패널** (차트 폭에 맞춤)
  - RSI, MACD, Revenue/Op.Profit 분기별 차트

#### 2.4 하단 정보 패널 (full-width 3칼럼)
- **Financials**: Revenue, Op.Profit, Net Margin, PER, PBR, ROE
- **AI Fusion**: 재무 + 기술 교차 분석, Buy/Sell signal, 태그
- **Sector Compare**: 동종 업종 대비 밸류에이션 + 모멘텀 비교 테이블

### 3. 파이프라인

에이전트 블록 오케스트레이션 시스템. **1개 종목 선택 → 파이프라인 실행 → 등록 시 백엔드 주기 실행** 구조.

- Topbar에 **종목 선택기** (1개 종목) + 파이프라인 드롭다운 + **Register & Run** 버튼
- 왼쪽 노드 팔레트에서 드래그하여 캔버스에 배치 (드래그 빌더 유지)
- 노드 카테고리: Agent Nodes / DSL Nodes / Output Nodes
- AND/OR 조건 분기 없음 — 각 노드가 독립 실행
- 하나의 파이프라인에 **3개 섹션**을 정의.

#### 3.1 에이전트 블록

- **모든 블록은 AI 에이전트** (Google ADK 기반)
- 에이전트는 목적에 따라 **도구를 자율적으로 선택**
- 블록 저장/불러오기
- 마켓플레이스에서 공유 가능

**에이전트 프롬프트 구조:**

각 블록의 프롬프트는 자유 텍스트가 아닌 **구조화된 필드**로 정의:

```
AgentBlockPrompt {
  name            string          // 블록 이름 (e.g. "뉴스 임팩트 분석")
  objective       string          // 목표 — 무엇을 해야 하는가
                                  // e.g. "이 종목의 최근 30일 뉴스를 분석하고
                                  //       산업 임팩트를 1~10점으로 평가"
  input_desc      string          // 입력 설명 — 어떤 데이터를 받는가
                                  // e.g. "종목 코드, 이벤트 날짜, 이전 블록의 분석 결과"
  tools           []string        // 사용 가능 도구 목록 (null = 전체)
                                  // e.g. ["search_news", "get_financials"]
  output_format   string          // 출력 형식 지시
                                  // e.g. "catalyst_type, impact_score(1-10),
                                  //       sentiment(positive/negative/neutral),
                                  //       key_findings 리스트"
  constraints     string?         // 제약사항
                                  // e.g. "미래 데이터 참조 금지, 추측 금지"
  examples        string?         // 예시 입출력 (few-shot)
}
```

**예시 — 뉴스 분석 블록:**

```yaml
name: "뉴스 임팩트 분석"
objective: |
  이 종목 관련 최근 30일 뉴스를 검색하고,
  가장 중요한 촉매를 식별한 뒤
  산업 관점에서 임팩트를 1~10점으로 평가해줘.
input_desc: "종목 코드, 종목명, 이벤트 날짜"
tools: ["search_news", "get_sector_stocks"]
output_format: |
  catalyst_type: 정책 | 실적 | M&A | 신사업 | 테마
  impact_score: 1~10
  sentiment: positive | negative | neutral
  key_findings: [{title, date, summary}]
  reasoning: 임팩트 점수의 근거
constraints: "뉴스 원문에 없는 내용을 추측하지 말 것"
examples: null
```

**예시 — 종목 탐지 블록:**

```yaml
name: "2년 최대거래량 탐지"
objective: |
  최근 5년 안에 2년 최대거래량이 발생하고
  거래대금이 3000억 이상인 종목을 찾아줘.
input_desc: "전체 종목 리스트 (또는 시장 필터)"
tools: ["dsl_evaluate", "scan_stocks", "get_candles"]
output_format: |
  matched_stocks: [{symbol, name, event_date, volume, trade_value}]
  total_count: 매칭 종목 수
constraints: null
examples: null
```

사용자는 이 필드들을 채워서 블록을 만들거나, **자연어로 설명하면 AI가 구조화된 프롬프트로 변환**해줄 수도 있음.

#### 3.2 파이프라인 3개 섹션

하나의 파이프라인 정의 안에 분석, 모니터링, 판단이 모두 포함:

```
Pipeline {
  ┌─────────────────────────────────────────────┐
  │ [분석 섹션] — 1회 실행                        │
  │   에이전트 블록들의 순차/병렬 오케스트레이션      │
  │   → 케이스 생성                               │
  ├─────────────────────────────────────────────┤
  │ [모니터링 섹션] — cron 반복 실행               │
  │   에이전트 블록 + 개별 cron 스케줄              │
  │   → 실행 결과를 타임라인에 기록                 │
  ├─────────────────────────────────────────────┤
  │ [판단 섹션] — DSL 경량 폴링                    │
  │   성공/실패 조건 + 가격 알림                    │
  │   → 1분 간격, LLM 호출 없음                   │
  └─────────────────────────────────────────────┘
}
```

- 사용자는 **하나의 파이프라인만** 만들면 됨
- 실행하면 분석 섹션이 1회 돌고, 모니터링 섹션이 자동으로 cron 등록
- 케이스 생성 후에도 모니터링 블록을 **추가/삭제/수정 가능**

#### 3.3 오케스트레이션 (분석 섹션)

- 에이전트 블록의 순차/병렬 실행 오케스트레이션
- **순차 실행**: 이전 에이전트의 출력이 다음 에이전트의 입력으로 전달
- **병렬 실행**: 같은 단계의 에이전트들이 동시에 실행, 모두 완료 후 다음 단계로
- 에이전트 간 데이터 전달 (컨텍스트 공유)
- 파이프라인 저장/불러오기
- 마켓플레이스에서 공유 가능

#### 3.4 파이프라인 구조 예시

```
[분석 섹션 — 1회 실행]

  [에이전트 A: 종목 탐지]
      │  "2년 최대거래량 + 3000억 탐지"
      │  출력: 종목 리스트
      │
      ├──────────────┬──────────────┐
      ▼              ▼              ▼        ← 병렬
  [에이전트 B]   [에이전트 C]   [에이전트 D]
   뉴스 분석     섹터 비교     재무 분석
      │              │              │
      ├──────────────┴──────────────┘
      ▼                                      ← 순차
  [에이전트 E: 종합 판단]
      ▼
  [케이스 생성 → 모니터링 섹션 cron 등록]

[모니터링 섹션 — cron 반복]

  "관련 뉴스 분석"              매 6시간
  "DART 공시 체크"              매일 18:00
  "섹터 동반 움직임 분석"        매일 장마감 후
  "산업 트렌드 변화"            매주 월요일
  → 각 실행 결과 → 케이스 타임라인에 기록

[판단 섹션 — DSL 경량 폴링]

  success = close >= event_high * 2.0
  failure = close < pre_event_ma(120)
  alert: "65,000원 도달" = close >= 65000
  → 1분 간격 체크, LLM 호출 없음
```

#### 3.4 성공/실패 판단 (DSL 스크립트)

파이프라인 실행 후, 케이스의 성공/실패를 판단하는 조건은 **DSL 스크립트**로 정의.
가격 기반 판단. 기간 제한 없음. 먼저 도달하는 쪽이 결과.

**이벤트 상대 변수 (초기 예시, 구현 시 확장 가능):**

| 변수 | 의미 |
|------|------|
| `event_high` | 이벤트 발생일 고가 |
| `event_low` | 이벤트 발생일 저가 |
| `event_close` | 이벤트 발생일 종가 |
| `event_volume` | 이벤트 발생일 거래량 |
| `pre_event_ma(N)` | 이벤트 전일 기준 N일 이평선 |
| `pre_event_close` | 이벤트 전일 종가 |
| `post_high` | 이벤트 이후 최고가 |
| `post_low` | 이벤트 이후 최저가 |
| `days_since_event` | 이벤트 이후 경과일 |

**예시:**

```javascript
// 성공: 이벤트일 고점 대비 100% 상승
success = close >= event_high * 2.0

// 실패: 이벤트 발생 전 120이평선 아래로 하락
failure = close < pre_event_ma(120)
```

### 4. 모니터링

> **별도 페이지 없음** — Pipeline Builder Section 2에서 정의, Dashboard에서 상태 확인, Case Timeline에서 결과 확인.

파이프라인 실행 후 케이스를 지속 감시하는 시스템. **두 가지 레이어**로 구성.

#### 4.1 에이전트 모니터링 블록 (cron 기반)

파이프라인과 **동일한 에이전트 블록 시스템**을 사용하되, cron 스케줄로 반복 실행.
실행 결과는 케이스 타임라인에 자동 기록.

- 사용자가 케이스별로 모니터링 블록을 **자유롭게 추가/삭제/수정** 가능 (케이스 생성 후에도)
- 각 블록마다 개별 cron 스케줄 설정
- 파이프라인의 분석 블록과 독립 — 처음 분석과 다른 관점으로 모니터링 가능
- **개별 블록 단위 중단/재개**: 특정 모니터링 블록만 일시 중지하거나 다시 시작
- **전체 모니터링 중단/재개**: 케이스 전체의 모니터링을 일괄 중단/재개
- 중단 중에도 가격 폴링(판단 섹션)은 선택적으로 유지 가능

**예시:**

| 모니터링 블록 | 스케줄 |
|-------------|--------|
| "관련 뉴스 분석하고 호재/악재 판단해줘" | 매 6시간 |
| "DART 공시 확인하고 중요 공시 알려줘" | 매일 18:00 |
| "동일 섹터 종목들 동반 움직임 분석해줘" | 매일 장마감 후 |
| "산업 트렌드 변화 체크해줘" | 매주 월요일 |

#### 4.2 가격 모니터링 (DSL 경량 폴링)

가격 관련 체크는 LLM 호출 없이 **DSL 엔진으로 경량 폴링**.

- **성공/실패 조건 체크**: DSL 스크립트 기반, 장중 1분 간격
  - 성공 조건 도달 → 케이스 종료 (성공) + 알림
  - 실패 조건 도달 → 케이스 종료 (실패) + 알림
- **사용자 지정 가격 알림**: 특정 가격 또는 기술적 조건(이평선 터치 등) 도달 시 기록 + 알림
- KIS API rate limit 고려: 종목별 요청을 배치로 묶어 처리

#### 4.3 정리

```
모니터링 = 에이전트 블록 (cron) + 가격 폴링 (DSL)

[에이전트 블록] → LLM 호출 → 정성 분석 (뉴스, 공시, 섹터)
                → cron 스케줄로 실행
                → 사용자가 자유롭게 편집

[가격 폴링]    → DSL 엔진 → 정량 체크 (성공/실패, 가격 알림)
                → 1분 간격 경량 폴링
                → LLM 비용 없음
```

- 모든 이벤트는 케이스 타임라인에 자동 축적

### 5. 알림

모니터링 이벤트 발생 시 사용자에게 알림. Pipeline Builder Section 3 (Alert & Notify)에서 채널 설정.

- 뉴스/공시/섹터 이벤트 알림
- 성공/실패 조건 도달 알림
- 사용자 지정 가격 도달 알림
- 알림 채널:
  - **In-App Alert**: Dashboard & Alerts 페이지에 표시
  - **Push Notification**: 모바일 푸시 알림
  - **Slack / Telegram**: 메신저 채널 연동

### 6. 케이스 관리

파이프라인 실행 결과를 **케이스**로 관리.

#### 6.1 케이스 = 리서치 파일

- 파이프라인 실행 결과로 생성
- 이벤트 발생일부터 종료(성공/실패)까지의 모든 기록을 담는 단위
- 상태: `LIVE` (모니터링 중) / `CLOSED:SUCCESS` / `CLOSED:FAILURE`

#### 6.2 Case Timeline 레이아웃

```
┌──────────────────────────────────────────────────────┐
│ [247540 ecoprobm LIVE] [005930 Samsung] [000660 SK..] │ ← 케이스 탭
│ 247540 ecoprobm  LIVE  D+127  -18.4%  Peak +42.1%   │ ← 선택된 케이스 요약
├──────────────────────────┬───────────────────────────┤
│ TIMELINE (50%)           │ DETAIL (50%)              │
│                          │                           │
│ ● D-Day: 2yr Max Volume  │ SUCCESS / FAILURE 조건    │
│ ● D+7: Foreign buying    │ Return Tracking 테이블    │
│ ● D+12: BUY 200주       │ Trade History             │
│ ● D+34: Peak +42.1%     │ Price Alerts              │
│ ● D+127: Current -18.4% │                           │
└──────────────────────────┴───────────────────────────┘
```

- **헤더**: 케이스 탭 바 (가로 스크롤) + 선택된 케이스 요약 정보
- **좌측 50%**: 타임라인 (D-Day부터 시간순 이벤트 축적)
- **우측 50%**: Success/Failure 조건 + Return Tracking + Trade History + Price Alerts

#### 6.3 타임라인

- 이벤트 발생일(D-Day)부터 시간순으로 기록 축적
- 각 기록: 날짜, 이벤트 유형(뉴스/공시/섹터/가격), 내용, AI 분석
- 수익률 추적: D+1, D+7, D+30, D+60... 시점별 수익률

#### 6.3 매수/매도 히스토리

- 사용자가 실제 매수/매도한 기록을 케이스에 연결
- 언제, 얼마에, 몇 주 매수/매도했는지 기록
- 파이프라인 판단 vs 실제 행동 비교
- 실현 손익 자동 계산
- 포트폴리오(Feature 8)와 자동 연동

#### 6.4 저장/불러오기/비교

- 케이스 저장 및 검색 (종목, 섹터, 촉매 유형, 결과 등으로 필터)
- 케이스 간 비교 (같은 전략의 다른 종목 결과 비교)
- 케이스 라이브러리에서 과거 케이스 참조

### 7. 백테스트

동일 파이프라인을 **과거 시점에 적용**하여 검증.

#### 7.1 LIVE 모드 vs BACKTEST 모드

- **같은 파이프라인 엔진**, 시점만 다름
- LIVE: 현재 시점에서 실행 → 모니터링 시작
- BACKTEST: 과거 시점에서 실행 → 이후 데이터로 결과 검증

#### 7.2 백테스트 실행

- 사용자가 기간 설정 (예: 2020-01 ~ 2025-12)
- 해당 기간 내 모든 이벤트 발생 건을 자동 탐지
- 각 이벤트에 대해 파이프라인 실행 + 성공/실패 판단
- 결과를 케이스로 생성 (상태: BACKTEST)

#### 7.3 통계

- 총 이벤트 수, 승률, 평균 수익률
- 섹터별 승률 및 성과
- 촉매 유형별 성과 (정책/실적/M&A/테마)
- 누적 수익률 차트 (전략 vs KOSPI 벤치마크)
- 최대 수익/최대 손실 케이스
- AI가 전략 개선 제안

#### 7.4 패턴 매칭

- 현재 LIVE 케이스와 과거 유사 케이스를 통계적으로 비교
- 매칭 기준: 섹터, 시가총액 범위, 촉매 유형, 섹터 동반 움직임 정도
- 매칭 결과:
  - 유사 케이스 N건 중 상승 확률 X%
  - 평균 수익률, 최대 낙폭, 최고점 도달 평균 일수
  - 유사도 상위 케이스 개별 비교
  - AI 패턴 인사이트 (과거 패턴 기반 현재 전망)

### 8. 포트폴리오

케이스의 매수/매도 히스토리와 연동되는 포트폴리오 관리.

- **자동 연동**: 케이스에 기록된 매수/매도가 포트폴리오에 자동 반영
- 실현/미실현 손익 분리
- 세금/수수료 포함 실질 수익률 계산
  - 국내/해외 주식 양도세 차등 적용
  - 수수료, 슬리피지 반영
- 섹터 비중 시각화
- 리밸런싱 시뮬레이터 (목표 비중 vs 현재 비중)

### 9. 마켓플레이스

파이프라인, 에이전트 블록, 검색 조건을 공유/거래하는 생태계.

- **공유 단위**: 파이프라인 전체, 개별 에이전트 블록, 검색 조건, 판단 스크립트
- 다른 사용자의 파이프라인을 가져와서 커스텀 가능
- 인기/성과 기반 랭킹
- 백테스트 결과를 함께 공유 (검증된 전략)

---

## Data Models

### Case (케이스)

```
Case {
  id              UUID
  user_id         UUID
  pipeline_id     UUID
  symbol          string          // 종목 코드 (e.g. "005930")
  status          enum            // LIVE | CLOSED_SUCCESS | CLOSED_FAILURE | BACKTEST
  event_date      date            // 이벤트 발생일 (D-Day)
  event_snapshot  EventSnapshot   // 이벤트 발생 시점의 스냅샷
  success_script  string          // 성공 조건 DSL
  failure_script  string          // 실패 조건 DSL
  closed_at       date?           // 종료일 (성공/실패 도달일)
  closed_reason   string?         // 종료 사유
  created_at      timestamp
  updated_at      timestamp
}

EventSnapshot {
  high            float64         // event_high
  low             float64         // event_low
  close           float64         // event_close
  volume          int64           // event_volume
  trade_value     int64           // 거래대금
  pre_ma          map[int]float64 // pre_event_ma(N) → 값 (N: 5,20,60,120,200)
}
```

### Timeline Event (타임라인 이벤트)

```
TimelineEvent {
  id              UUID
  case_id         UUID
  date            date
  type            enum            // NEWS | DISCLOSURE | SECTOR | PRICE_ALERT | TRADE | PIPELINE_RESULT
  title           string
  content         string          // 상세 내용
  ai_analysis     string?         // AI 분석 요약
  data            json?           // 이벤트 유형별 구조화 데이터
  created_at      timestamp
}
```

### Trade (매수/매도 기록)

```
Trade {
  id              UUID
  case_id         UUID
  user_id         UUID
  type            enum            // BUY | SELL
  price           float64
  quantity        int
  fee             float64         // 수수료
  date            date
  note            string?
  created_at      timestamp
}
```

### Pipeline (파이프라인)

```
Pipeline {
  id              UUID
  user_id         UUID
  name            string
  description     string

  // 분석 섹션 — 1회 실행
  analysis_stages []Stage         // 순서가 있는 단계 목록

  // 모니터링 섹션 — cron 반복
  monitors        []MonitorBlock  // 모니터링 블록 + 스케줄

  // 판단 섹션 — DSL 경량 폴링
  success_script  string          // 성공 조건 DSL
  failure_script  string          // 실패 조건 DSL
  price_alerts    []PriceAlert    // 사용자 지정 가격 알림

  is_public       bool            // 마켓플레이스 공개 여부
  created_at      timestamp
  updated_at      timestamp
}

Stage {
  order           int             // 실행 순서 (같은 order = 병렬)
  blocks          []AgentBlock    // 이 단계의 에이전트 블록들
}

MonitorBlock {
  block           AgentBlock      // 에이전트 블록 (동일한 블록 시스템)
  cron            string          // cron 표현식 (e.g. "0 */6 * * *")
  enabled         bool            // 중단/재개 제어
}
```

### Agent Block (에이전트 블록)

```
AgentBlock {
  id              UUID
  user_id         UUID
  name            string
  instruction     string          // 자연어 지시문 (에이전트의 역할 정의)
  system_prompt   string?         // 추가 시스템 프롬프트 (선택)
  allowed_tools   []string?       // 사용 가능 도구 제한 (null = 전체)
  output_schema   json?           // 기대 출력 형식 (선택)
  is_public       bool
  created_at      timestamp
}
```

### Price Alert (사용자 지정 가격 알림)

```
PriceAlert {
  id              UUID
  case_id         UUID
  condition       string          // DSL 표현식 (e.g. "close >= 75000")
  label           string          // 사용자 메모 (e.g. "목표가 도달")
  triggered       bool
  triggered_at    date?
}
```

---

## Agent I/O Contract

### Agent Block 입출력

모든 에이전트 블록은 동일한 입출력 구조를 따른다:

**입력 (AgentInput):**
```json
{
  "instruction": "이 종목의 최근 30일 뉴스를 분석하고 산업 임팩트를 평가해줘",
  "context": {
    "symbol": "247540",
    "symbol_name": "에코프로비엠",
    "event_date": "2025-11-12",
    "event_snapshot": { "high": 152000, "volume": 28400000, ... },
    "previous_results": [
      {
        "block_name": "종목 탐지",
        "summary": "2년 최대거래량 발생. 거래대금 4,280억.",
        "data": { "matched_stocks": ["247540", "373220"] }
      }
    ]
  }
}
```

**출력 (AgentOutput):**
```json
{
  "summary": "IRA 보조금 최종 확정으로 양극재 수혜 직접 대상. 산업 임팩트 8/10.",
  "data": {
    "catalyst_type": "정책",
    "impact_score": 8,
    "sentiment": "positive",
    "key_articles": [...]
  },
  "confidence": 0.85
}
```

### 데이터 전달 규칙

1. **순차 실행**: 이전 단계의 모든 AgentOutput이 다음 단계의 `context.previous_results`로 전달
2. **병렬 실행**: 같은 단계 블록들은 동일한 `context`를 받고, 각자 독립적으로 실행
3. **summary는 필수**: 모든 에이전트는 결과를 `summary` (자연어 요약)로 반드시 반환. 다음 에이전트가 이를 컨텍스트로 활용
4. **data는 선택**: 구조화 데이터가 있으면 `data` 필드에 JSON으로. 스키마는 블록의 `output_schema`로 선택적 강제 가능
5. **confidence는 선택**: 에이전트의 판단 신뢰도 (0.0~1.0)

### 백테스트 모드 에이전트 동작

백테스트에서 에이전트는 **과거 시점으로 제한된 컨텍스트**를 받는다:

- 에이전트에게 `backtest_date`가 전달됨: "당신은 2022-08-22 시점에서 분석하고 있습니다"
- 도구들은 해당 날짜 이전 데이터만 반환 (미래 데이터 차단)
- `search_news` → 해당 날짜 이전 뉴스만 반환 (뉴스 아카이브 필요)
- `get_financials` → 해당 날짜 기준 최신 공시 재무제표 반환
- `get_candles` → 해당 날짜까지의 캔들만 반환
- 성공/실패 판단은 이후 실제 가격 데이터로 계산 (에이전트 호출 없이 DSL로)

---

## API Endpoints

### 종목 검색
- `POST /api/search/scan` — 에이전트 기반 종목 스캔 실행
- `GET /api/search/presets` — 프리셋 검색 조건 목록
- `POST /api/search/presets` — 검색 조건 저장
- `DELETE /api/search/presets/:id` — 검색 조건 삭제

### 차트
- `GET /api/candles/:symbol` — 캔들 데이터 (기존 KIS 연동)
- `GET /api/financials/:symbol/events` — 재무 이벤트 마커 (실적발표, 유상증자 등)
- `GET /api/financials/:symbol/statements` — 재무제표 (DART)
- `GET /api/sector/:symbol/compare` — 섹터 내 비교 데이터

### 파이프라인
- `GET /api/pipelines` — 내 파이프라인 목록
- `POST /api/pipelines` — 파이프라인 생성
- `PUT /api/pipelines/:id` — 파이프라인 수정
- `DELETE /api/pipelines/:id` — 파이프라인 삭제
- `POST /api/pipelines/:id/execute` — 파이프라인 실행 (→ 케이스 생성)
- `GET /api/pipelines/:id/jobs/:jobId` — 실행 상태 조회

### 에이전트 블록
- `GET /api/blocks` — 내 블록 목록
- `POST /api/blocks` — 블록 생성
- `PUT /api/blocks/:id` — 블록 수정
- `DELETE /api/blocks/:id` — 블록 삭제

### 케이스
- `GET /api/cases` — 케이스 목록 (필터: status, symbol, sector)
- `GET /api/cases/:id` — 케이스 상세
- `DELETE /api/cases/:id` — 케이스 삭제
- `GET /api/cases/:id/timeline` — 타임라인 이벤트 목록
- `POST /api/cases/:id/trades` — 매수/매도 기록 추가
- `GET /api/cases/:id/trades` — 매수/매도 히스토리
- `POST /api/cases/:id/alerts` — 가격 알림 추가
- `GET /api/cases/:id/alerts` — 가격 알림 목록

### 백테스트
- `POST /api/backtest` — 백테스트 실행 (pipeline_id + 기간)
- `GET /api/backtest/jobs/:jobId` — 백테스트 진행 상태
- `GET /api/backtest/:id/stats` — 백테스트 통계
- `GET /api/backtest/:id/cases` — 백테스트 케이스 목록
- `POST /api/cases/:id/pattern-match` — 유사 케이스 패턴 매칭

### 포트폴리오
- `GET /api/portfolio` — 포트폴리오 현황
- `GET /api/portfolio/history` — 손익 히스토리
- `GET /api/portfolio/tax` — 세금 시뮬레이션

### 마켓플레이스
- `GET /api/marketplace/pipelines` — 공개 파이프라인 목록
- `GET /api/marketplace/blocks` — 공개 블록 목록
- `POST /api/marketplace/pipelines/:id/fork` — 파이프라인 복제
- `POST /api/marketplace/blocks/:id/fork` — 블록 복제

### 알림
- `GET /api/notifications` — 알림 목록
- `GET /api/notifications/stream` — SSE 실시간 알림 스트림
- `PUT /api/notifications/:id/read` — 읽음 처리

### 모니터링 (내부)
모니터링 서비스는 외부 API가 아닌 **백엔드 내부 워커**로 동작:
- 가격 체크: 장중 1분 간격 (KIS API 호출 최소화, 배치 조회)
- 뉴스/공시: 5분 간격 폴링
- 섹터 움직임: 장 마감 후 1회 일괄 계산
- KIS API rate limit 고려: 종목별 요청을 배치로 묶어 처리

---

## Architecture

### Data Sources

| 소스 | 용도 |
|------|------|
| KIS Open API | 실시간 가격, 캔들(일/주/월/분), 뉴스, 거래 데이터 |
| DART (전자공시) | 재무제표, 공시, 사업보고서, 대주주 변동 |
| KRX 정보데이터 | 섹터 분류, ETF 구성, 외국인/기관 매매 |
| 뉴스 크롤링 | 종목/섹터 뉴스, 소셜 센티먼트 |

### Backend Core

- **Agent Runtime**: LLM 에이전트 실행 환경. 도구 호출, 세션 관리
- **Pipeline Orchestrator**: 에이전트 블록의 순차/병렬 실행, 데이터 전달
- **Monitoring Service**: 백엔드 상주. 뉴스/공시/섹터/가격 감시, 이벤트 기록, 알림
- **DSL Engine**: 가격/거래량/지표 정량 계산, 성공/실패 조건 평가
- **Fundamental Engine**: 재무 분석, 밸류에이션, 섹터 비교
- **Case Store**: 케이스 CRUD, 타임라인, 매수/매도 히스토리
- **Backtest Engine**: 과거 시점 파이프라인 실행, 통계, 패턴 매칭

### Agent Tool Registry

에이전트가 사용할 수 있는 도구 목록. 각 도구는 **구조화된 스키마**로 정의되어 ADK에 등록.

#### Tool 정의 구조

```
AgentTool {
  name            string          // 도구 이름 (함수명)
  description     string          // 도구 설명 (에이전트가 이해할 수 있는)
  category        string          // 카테고리 (price, fundamental, news, sector, dsl)
  parameters      []ToolParam     // 입력 파라미터 정의
  returns         ToolReturn      // 반환값 정의
}

ToolParam {
  name            string
  type            string          // string, int, float, date, []string 등
  description     string
  required        bool
  default         any?
}

ToolReturn {
  type            string          // object, array, scalar
  schema          json            // 반환값 JSON 스키마
}
```

#### 초기 도구 목록 (구현 시 확장 가능)

**가격/차트 도구:**

| 도구 | 설명 | 주요 파라미터 | 반환 |
|------|------|-------------|------|
| `get_candles` | 캔들 데이터 조회 | symbol, timeframe, from, to | [{date, open, high, low, close, volume}] |
| `get_price` | 현재가 조회 | symbol | {price, change, change_pct, volume, per, eps} |
| `scan_stocks` | 조건 기반 종목 스캐닝 | market, dsl_expression | [{symbol, name, matched_value}] |

**재무/회계 도구:**

| 도구 | 설명 | 주요 파라미터 | 반환 |
|------|------|-------------|------|
| `get_financials` | 재무제표 조회 (DART) | symbol, year, quarter | {revenue, operating_profit, net_income, ...} |
| `get_disclosures` | 공시 목록 조회 (DART) | symbol, from, to, type | [{title, date, type, url}] |
| `get_valuation` | 밸류에이션 지표 | symbol | {per, pbr, roe, ev_ebitda, dividend_yield} |

**뉴스/센티먼트 도구:**

| 도구 | 설명 | 주요 파라미터 | 반환 |
|------|------|-------------|------|
| `search_news` | 뉴스 검색 및 분석 | symbol, keyword, from, to, limit | [{title, date, source, summary, sentiment}] |

**섹터/시장 도구:**

| 도구 | 설명 | 주요 파라미터 | 반환 |
|------|------|-------------|------|
| `get_sector_stocks` | 동일 섹터 종목 목록 | symbol 또는 sector_code | [{symbol, name, market_cap}] |
| `compare_sector` | 섹터 내 상대 비교 | symbol, metrics[] | [{symbol, per, roe, rsi, rank}] |
| `get_fund_flow` | 외국인/기관 매매 동향 | symbol, from, to | [{date, foreign_net, institution_net}] |

**DSL 도구:**

| 도구 | 설명 | 주요 파라미터 | 반환 |
|------|------|-------------|------|
| `dsl_evaluate` | DSL 표현식 평가 | symbol, expression, from, to | {result, series?} |

### Frontend

- 파이프라인 빌더 (에이전트 블록 조합 UI)
- 차트 뷰 (기술적 지표 + 재무 이벤트 오버레이)
- 케이스 타임라인 뷰
- 백테스트 결과 대시보드
- 패턴 매칭 비교 뷰
- 포트폴리오 대시보드
- 마켓플레이스 브라우저
- 알림 센터

### Tech Stack (예상)

- **Frontend**: Next.js, React, TailwindCSS, lightweight-charts
- **Backend**: Go (Gin), PostgreSQL
- **Agent Runtime**: Google ADK (Agent Development Kit) — Tool Calling, 세션 관리, 에이전트 오케스트레이션
- **Data**: KIS API, DART API, KRX
- **Monitoring**: Background worker (Go goroutines)
- **Auth**: Google OAuth, JWT (httpOnly cookie)

---

## Design Decision Log

| 결정 | 근거 |
|------|------|
| 모든 블록을 에이전트로 | DSL 스크립트만으로는 뉴스 해석, 산업 임팩트 평가 불가. 에이전트가 도구를 자율 선택하면 정량+정성 분석 모두 커버 |
| 파이프라인 1회 실행 + 모니터링 분리 | 파이프라인 재실행은 비용과 지연이 큼. 모니터링은 경량 백엔드 서비스로 상시 동작 |
| 성공/실패 조건을 DSL 스크립트로 | 단순 퍼센트가 아닌 이벤트 맥락(이벤트일 고가, 이벤트 전 이평선 등)을 참조해야 하므로 스크립트 필요 |
| 판단 기준은 가격만, 기간 무관 | 사용자 요구. 먼저 도달하는 쪽(성공 또는 실패)이 결과 |
| 팀 협업 기능 제외 | 사용자 요청으로 스코프에서 제거 |
| 완전 새출발 (기존 코드 참고만) | 패러다임이 근본적으로 다르므로 기존 ChartingLens 코드베이스 위에 얹기 어려움 |
| 수동 트리거 (자동매매 없음) | 사용자가 분석 결과를 직접 보고 판단. 자동 주문 실행 없음 |
| 백테스트 시 에이전트도 실행 | 과거 뉴스/공시 해석이 핵심이므로 에이전트 호출 필요. 단, 미래 데이터 차단 + 뉴스 아카이브 필수 |
| 모니터링은 경량 폴링 | 에이전트 재실행이 아닌 데이터 폴링 (가격 1분, 뉴스 5분, 섹터 장마감 후). KIS rate limit 준수 |
| Agent I/O는 summary+data 구조 | summary(자연어)로 에이전트 간 컨텍스트 전달, data(JSON)로 구조화 데이터 선택적 전달 |

---

## Glossary

| 용어 | 정의 |
|------|------|
| **파이프라인** | 에이전트 블록들의 순차/병렬 실행 흐름 정의. 재사용 가능한 전략 단위 |
| **블록** | 파이프라인의 구성 단위. 하나의 AI 에이전트가 하나의 분석 작업을 수행 |
| **케이스** | 파이프라인 실행 결과로 생성되는 리서치 파일. 이벤트 발생부터 종료까지의 모든 기록 |
| **이벤트** | 케이스 내 타임라인에 기록되는 개별 사건 (뉴스, 공시, 가격 변동 등) |
| **모니터링** | 백엔드에서 LIVE 케이스를 지속 감시하는 서비스. 파이프라인 재실행이 아닌 데이터 폴링 |
| **패턴 매칭** | 현재 케이스와 과거 유사 케이스를 통계적으로 비교하는 기능 |
| **DSL** | 성공/실패 조건을 정의하는 스크립트 언어. 이벤트 상대 변수 지원 |
