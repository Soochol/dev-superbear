---
name: verify-fsd
description: FSD(Feature-Sliced Design) 아키텍처 규칙 검증. 위젯/피처/엔티티 구조 변경, 새 슬라이스 추가, import 경로 수정 후 사용.
---

# FSD 아키텍처 검증

## Purpose

1. **Public API (barrel file) 규칙** — 모든 슬라이스가 `index.ts` barrel 파일을 통해서만 외부에 노출되는지 검증
2. **Import 경계 규칙** — 외부 소비자가 슬라이스 내부 경로(`/ui/`, `/model/`, `/lib/`)를 직접 import하지 않는지 검증
3. **Store 위치 규칙** — Zustand store가 올바른 FSD 레이어의 `model/` 디렉토리에 위치하는지 검증
4. **Shared 레이어 구조** — `shared/` 디렉토리가 허용된 서브디렉토리만 포함하는지 검증
5. **디자인 토큰 네이밍** — CSS 커스텀 프로퍼티가 `--color-nexus-*` 또는 `--font-*` 네이밍 규칙을 따르는지 검증

## When to Run

- 새 widget, feature, entity 슬라이스를 추가한 후
- 기존 슬라이스의 import 경로를 변경한 후
- `shared/` 레이어에 새 파일이나 디렉토리를 추가한 후
- `globals.css`에 새 디자인 토큰을 추가한 후
- FSD 구조와 관련된 리팩토링 후

## Related Files

| File | Purpose |
|------|---------|
| `src/widgets/*/index.ts` | Widget barrel 파일 (public API) |
| `src/features/*/index.ts` | Feature barrel 파일 (public API) |
| `src/entities/*/index.ts` | Entity barrel 파일 (public API) |
| `src/shared/model/*.store.ts` | Global Zustand store |
| `src/shared/config/constants.ts` | 앱 전역 상수 |
| `src/app/globals.css` | 디자인 토큰 (CSS 커스텀 프로퍼티) |

## Workflow

### Step 1: Widget barrel 파일 존재 확인

**검사:** 모든 `src/widgets/*/` 디렉토리에 `index.ts` barrel 파일이 있는지 확인합니다.

```bash
for dir in src/widgets/*/; do
  widget=$(basename "$dir")
  if [ ! -f "${dir}index.ts" ]; then
    echo "FAIL: src/widgets/$widget/index.ts 없음"
  fi
done
```

**PASS 기준:** 모든 widget 디렉토리에 `index.ts`가 존재
**FAIL 기준:** `index.ts`가 없는 widget 디렉토리가 있음

**수정:** 해당 widget의 주요 컴포넌트를 export하는 `index.ts`를 생성합니다. 내부 서브컴포넌트는 export하지 않습니다.

### Step 2: Feature/Entity barrel 파일 존재 확인

**검사:** 모든 `src/features/*/`, `src/entities/*/` 디렉토리에 `index.ts`가 있는지 확인합니다.

```bash
for dir in src/features/*/ src/entities/*/; do
  slice=$(echo "$dir" | sed 's|src/||;s|/$||')
  if [ ! -f "${dir}index.ts" ]; then
    echo "FAIL: src/$slice/index.ts 없음"
  fi
done
```

**PASS 기준:** 모든 feature/entity 디렉토리에 `index.ts`가 존재
**FAIL 기준:** `index.ts`가 없는 디렉토리가 있음

**수정:** 해당 슬라이스의 public API를 export하는 `index.ts`를 생성합니다.

### Step 3: Widget 내부 경로 직접 import 탐지

**검사:** widget 외부 코드가 `@/widgets/{name}/ui/` 또는 `@/widgets/{name}/model/`을 직접 import하는지 탐지합니다.

Grep 도구 사용:
- **pattern:** `from.*@/widgets/[^"']+/(?:ui|model|lib)/`
- **path:** `src/`
- **glob:** `*.{ts,tsx}`

탐지된 각 import에 대해:
1. import하는 파일이 **같은 widget 내부**에 있는지 확인 (내부 import는 허용)
2. 외부 파일에서의 import만 FAIL로 보고

**PASS 기준:** widget 외부에서 내부 경로 직접 import가 없음
**FAIL 기준:** `@/widgets/{name}/ui/Component` 같은 내부 경로를 외부에서 직접 import

**수정:** barrel 파일(`@/widgets/{name}`)을 통해 import하도록 변경합니다. barrel 파일에 해당 export가 없으면 추가합니다.

### Step 4: Feature/Entity 내부 경로 직접 import 탐지

**검사:** 슬라이스 외부 코드가 `@/features/{name}/...` 또는 `@/entities/{name}/...` 내부 경로를 직접 import하는지 탐지합니다.

Grep 도구 사용:
- **pattern:** `from.*@/(features|entities)/[^"']+/(?:ui|model|lib|api)/`
- **path:** `src/`
- **glob:** `*.{ts,tsx}`

탐지된 각 import에 대해:
1. import하는 파일이 **같은 슬라이스 내부**에 있는지 확인 (내부 import는 허용)
2. 외부 파일에서의 import만 FAIL로 보고

**PASS 기준:** 슬라이스 외부에서 내부 경로 직접 import가 없음
**FAIL 기준:** `@/features/{name}/model/store` 같은 내부 경로를 외부에서 직접 import

**수정:** barrel 파일(`@/features/{name}` 또는 `@/entities/{name}`)을 통해 import하도록 변경합니다.

### Step 5: Zustand store 위치 검증

**검사:** Zustand store 파일(`create<...>()` 호출이 있는 파일)이 올바른 FSD `model/` 디렉토리에 위치하는지 확인합니다.

Grep 도구 사용:
- **pattern:** `create<.*>\(`
- **path:** `src/`
- **glob:** `*.ts`

허용되는 위치 패턴:
- `src/shared/model/*.store.ts`
- `src/features/*/model/*.store.ts`
- `src/entities/*/model/*.store.ts`

**PASS 기준:** 모든 store 파일이 위 패턴에 해당하는 경로에 위치
**FAIL 기준:** `model/` 디렉토리 외부에 store가 있거나, `widgets/` 또는 `app/` 레이어에 store가 있음

**수정:** store를 적절한 FSD 레이어의 `model/` 디렉토리로 이동합니다.

### Step 6: Shared 레이어 구조 검증

**검사:** `src/shared/` 직하위에 허용된 디렉토리만 있는지 확인합니다.

```bash
for dir in src/shared/*/; do
  name=$(basename "$dir")
  case "$name" in
    api|config|lib|model|ui) ;; # 허용
    __tests__|__mocks__) ;; # 테스트 관련 허용
    *) echo "FAIL: src/shared/$name/ — 허용되지 않은 디렉토리" ;;
  esac
done
```

허용되는 서브디렉토리: `api/`, `config/`, `lib/`, `model/`, `ui/`, `__tests__/`, `__mocks__/`

**PASS 기준:** 허용된 디렉토리만 존재
**FAIL 기준:** 허용되지 않은 서브디렉토리가 존재

**수정:** 허용되지 않은 디렉토리를 적절한 FSD 레이어(features, entities)로 이동하거나 허용된 디렉토리에 통합합니다.

### Step 7: 디자인 토큰 네이밍 검증

**검사:** `src/app/globals.css`의 `@theme inline` 블록 내 CSS 커스텀 프로퍼티가 네이밍 규칙을 따르는지 확인합니다.

Grep 도구 사용:
- **pattern:** `--[a-z]`
- **path:** `src/app/globals.css`
- **output_mode:** `content`

허용되는 접두사:
- `--color-nexus-*` — 색상 토큰
- `--font-*` — 폰트 토큰

**PASS 기준:** 모든 커스텀 프로퍼티가 허용된 접두사로 시작
**FAIL 기준:** `--color-nexus-`나 `--font-`로 시작하지 않는 커스텀 프로퍼티가 있음

**수정:** 적절한 네이밍 접두사로 변경합니다. 새로운 시맨틱 카테고리가 필요한 경우 팀과 논의합니다.

## Output Format

```markdown
## FSD 아키텍처 검증 결과

| # | 검사 항목 | 상태 | 상세 |
|---|----------|------|------|
| 1 | Widget barrel 파일 | PASS / N개 이슈 | 상세... |
| 2 | Feature/Entity barrel 파일 | PASS / N개 이슈 | 상세... |
| 3 | Widget 내부 import | PASS / N개 이슈 | 상세... |
| 4 | Feature/Entity 내부 import | PASS / N개 이슈 | 상세... |
| 5 | Store 위치 | PASS / N개 이슈 | 상세... |
| 6 | Shared 레이어 구조 | PASS / N개 이슈 | 상세... |
| 7 | 디자인 토큰 네이밍 | PASS / N개 이슈 | 상세... |
```

## Exceptions

다음은 **위반이 아닙니다**:

1. **슬라이스 내부의 상대 import** — 같은 widget/feature/entity 내에서 `./ui/Component`나 `../model/types` 같은 내부 import는 허용됩니다. barrel 파일은 외부 소비자를 위한 것입니다.
2. **Type-only import** — `import type { ... } from '@/entities/case/model/types'` 같은 타입 전용 import는 barrel 파일이 없는 레거시 슬라이스에서 일시적으로 허용됩니다. 단, barrel 파일이 생성되면 이를 통해 import해야 합니다.
3. **테스트 파일의 내부 import** — `__tests__/` 디렉토리 내의 테스트 파일은 테스트 대상 모듈의 내부 경로를 직접 import할 수 있습니다.
4. **Tailwind/PostCSS 생성 변수** — `@theme inline` 블록 외부에서 Tailwind가 자동 생성하는 CSS 변수는 검증 대상이 아닙니다.
5. **Mock 파일의 import** — `__mocks__/` 디렉토리의 파일은 mock 대상 모듈의 내부 구조를 참조할 수 있습니다.
