# Baton 한국어 가이드

이 문서는 Baton을 프로젝트에 도입하거나 운영할 때 읽는 한국어 해설서입니다.
정식 배포 파일은 `bootstrap/` 아래에 있으며, 이 문서는 그 파일들의 의도와 사용 방법을 설명합니다.

## 1. Baton란

Baton은 **Director / Planner / Executor 에이전트 팀**이 역할을 나누고, 기록 기반으로 작업을 이어가기 위한 파일 기반 협업 규약입니다. 프로젝트 안에 작업 분류, 이벤트 타임라인, 라운드 산출물, 작업 맥락 전달 표준을 남기는 것입니다.

Baton은 macOS/Linux/Windows의 amd64·arm64용 단일 Go 바이너리로
배포하며 설치된 프로젝트에는 Go 런타임이나 특정 셸이 필요하지 않습니다.
Windows에서는 PowerShell, cmd, Git Bash에서 실행할 수 있습니다. Git 연동
기능을 사용하려면 `git`이 `PATH`에 있어야 합니다.

## 2. 에이전트 팀 구성

저장소 수준 지시가 다른 절차를 지정하지 않는 한, 표준 구현 작업에는 먼저 **Director / Planner / Executor 에이전트 팀**을 구성하고 Director / Planner / Executor 프로토콜을 적용합니다. **Director (허브)**는 사용자 소통, 분류, 범위/위험 결정, 위임, 결과 해석, 최종 보고를 담당합니다. Director는 Planner/Executor에게 백그라운드로 작업을 위임한 뒤 즉시 짧은 상태를 사용자에게 반환하고, 위임 중에도 새 사용자 요구에 대응 가능한 상태를 유지합니다. Planner와 Executor는 Director를 통해서만 통신합니다.

| 역할 | 책임 |
| --- | --- |
| **Director (허브)** | 작업을 라우팅하고 증거가 요청을 충족하는지 판단합니다. 단순 중계자가 아닙니다. |
| **Planner** | `PLAN`을 작성하고, 구현이 계획과 일치하는지 검토 증거를 정리합니다. `REVIEW`는 승인이 아니며 발견은 `blocker`(반드시 수정) 또는 `nit`(비차단)으로 표시합니다. |
| **Executor** | `PLAN`을 구현하고 검증합니다. 모호함은 범위를 넓히지 않은 채 Director에게 되돌립니다. |

Planner와 Executor는 **Director를 통해서만** 통신합니다. 사용 도구의 능력에 따라 배정된 멤버를 병렬 또는 순차로 실행할 수 있지만, 기록 없이 단일 에이전트 작업으로 축소해서는 안 됩니다.

**강제 선행 규칙:** 기록이 필요한 각 단계의 담당 역할은 단계 시작 시 `.baton/GUIDANCE.md`와 `.baton/LESSON-LEARNED.md` 인덱스를 읽고, 현재 범위와 `Applies When` 또는 `Trigger / Symptom`이 맞는 개별 기록만 `.baton/lesson-learned/`에서 읽습니다. Director의 초기 선별만 의존하지 않고, Planner와 Executor도 자신의 확장된 단계 범위에 맞춰 다시 선별합니다. 명백한 기록 제외 요청은 이 확인 없이 응답할 수 있습니다.

## 3. 작업 분류

새 세션을 시작할 때마다 Director는 먼저 Codex 또는 Claude Code를 감지하고 Director·Planner·Executor 모델을 묻습니다. `baton models get <platform>`의 직전 선택을 기본값으로, `baton models list <platform>`의 현재 목록을 선택지로 보여 줍니다. Director 모델이 현재 모델과 다르면 사용자가 `/model`로 바꾸고 `/status`로 확인한 뒤 계속합니다. Planner와 Executor 모델 및 reasoning effort는 위임할 때 명시하며, 확정값은 `baton models set`으로 OS 사용자 설정에 플랫폼별 저장합니다. 그 다음 이번 Baton 세션에서 Git 브랜치 전략을 사용할지 묻습니다: 항상 브랜치 사용, 브랜치 사용 안 함, 작업마다 확인.

Director는 먼저 요청이 명백한 기록 제외 대상인지 가볍게 판단합니다. 기록이 필요한 요청이면 필수 지침·교훈 확인을 마친 뒤 `Solo` 또는 `Relay`로 분류합니다. Baton **부트스트랩**과 **업데이트**(`.baton/`·Baton 지시 파일 동기화)는 기록 제외가 아니며, Director가 직접 수행하면 `Solo`로 `REQUEST → RUN_DONE`을 기록합니다.

| 분류 | 일반적 범위 | 처리 방식 |
| --- | --- | --- |
| 기록 제외 | 단순 질문 답변, 짧은 설명, 브레인스토밍 | 응답만 하고 이벤트를 남기지 않음 |
| `Solo` | 사소한 텍스트/설정 변경, 명백한 국소 편집, Baton 부트스트랩·업데이트 | Director가 직접 처리하고 `REQUEST → RUN_DONE` 기록 |
| `Relay` | 다중 파일 구현, 설계 판단, 구현 검증이 필요한 작업 | 세션 브랜치 전략 적용 → Planner → Executor → Planner 검토 → 승인 후 필요 시 자동 병합 |

## 4. 백그라운드 위임

`Relay` 작업은 분류 직후 세션 시작 때 정한 Git 브랜치 전략을 따릅니다. 전용 작업 브랜치를 쓰는 경우 현재 브랜치를 기준 브랜치로 기억하고 작업 브랜치를 만든 뒤, 그 브랜치에서 `REQUEST`부터 기록합니다. 브랜치를 쓰지 않는 전략이면 현재 브랜치에서 기록과 변경을 진행합니다. 전용 작업 브랜치를 쓴 경우 승인 후 `CLOSE`을 기록해 승인 상태를 커밋한 다음 기준 브랜치로 자동 병합합니다. 위임은 가능한 한 백그라운드로 수행하며 Director는 위임 직후 사용자에게 짧은 상태를 반환하고 완료 대기·폴링·sleep으로 사용자 응답을 막지 않습니다. 하위 AI의 완료 통지에 의존하지 않고, 위임 직후 기대 산출물에 대해 `baton await <PLANNED|EXECUTED|REVIEW> <path>`를 백그라운드 명령으로 실행합니다(Claude Code에서는 `run_in_background: true`). 산출물이 `check-artifact`를 통과하면 0으로 종료되어 호스트가 Director를 깨우고, Director는 해당 이벤트를 기록합니다. 0이 아닌 종료는 제한 시간(기본 30분) 초과이므로 하위 AI 상태를 확인합니다. 사용자와 대화하는 도중 `await` 종료를 받으면 진행 중인 응답을 마친 직후, 다른 작업보다 먼저 이벤트 기록 → gate 확인 → 다음 단계 위임을 수행합니다. 통지를 놓친 경우의 안전망으로, Relay 작업이 열려 있는 동안 Director는 사용자 메시지마다 `baton status`를 한 번 실행합니다. `pending_artifact:` 줄은 완성됐지만 아직 기록되지 않은 산출물을 뜻하며 같은 방식으로 처리합니다.

## 5. 이벤트 타임라인

`BATON-LOG.txt`는 추가 전용 이벤트 타임라인입니다. 기존 줄을 수정하지 않습니다.

```text
<YYYY-MM-DDTHH:MM:SS> | <task-id> | <event> | <role> | <summary> | <path?>
```

- `timestamp`는 로컬 시스템 시간 기준 `YYYY-MM-DDTHH:MM:SS` 형식으로 기록합니다.
- `task-id`는 무작위 소문자 영문 4글자를 씁니다.
- Director는 `REQUEST` 기록 시 `task-id` 하나를 정하고, 같은 Relay 작업의 `PLANNED`/`EXECUTED`/`REVIEW`/`FEEDBACK`/`CLOSE`까지 재사용합니다. 새 `REQUEST`마다 새 `task-id`를 씁니다.
- 이벤트는 `REQUEST`, `PLANNED`, `EXECUTED`, `REVIEW`, `FEEDBACK`, `CLOSE`, `RUN_DONE`만 씁니다.
- Director 직접 처리 흐름은 `REQUEST → RUN_DONE`입니다.
- 표준 처리 흐름은 `REQUEST` → `PLANNED` → `EXECUTED` → `REVIEW` → `CLOSE`입니다.
- `Relay`의 `REQUEST`는 세션 브랜치 전략을 적용한 뒤 기록합니다. 전용 작업 브랜치를 쓰는 경우 승인 전에는 기준 브랜치에 해당 작업의 이벤트나 변경을 기록하지 않습니다.
- `FEEDBACK`은 사용자가 `CLOSE` 승인 전 피드백·결함을 알려줄 때 Director가 기록합니다. 같은 `task-id`와 산출물 파일 키를 유지합니다.
- 사용자가 보고한 결함은 Director가 Executor에 전달하고, 수정 전 증거와 수정 후 셀프 스모크 테스트를 `RUN`에 기록합니다.
- `FEEDBACK` 후 Director는 **현재 런에 추가**할지 **새로운 런**으로 돌릴지 사용자에게 묻습니다. 명백한 결함이면 사용자 확인 없이 현재 런에 추가합니다.
- **현재 런에 추가**: 마지막 `RUN-<NN>` 범위와 기존 `PLAN` 안에서 `EXECUTED` → `REVIEW-<NN>`을 다시 진행합니다. `CLOSE` 이벤트 승인 전이면 `RUN-<NN>.md` 갱신을 허용합니다.
- **새로운 런**: 다음 `RUN-<NN+1>`로 `EXECUTED` → `REVIEW-<NN+1>`을 진행합니다.
- `role` 주변 공백은 정렬용이며 의미가 없습니다.
- `event`는 8자 폭, `role`은 8자 폭으로 왼쪽 정렬하고 부족한 자리는 공백으로 채웁니다.
- `EXECUTED`는 Executor가 `RUN-<NN>.md`를 작성한 뒤 Director가 기록합니다. `path` 필수입니다.
- 한 라운드 `<NN>`은 하나의 `EXECUTED`와 그에 대응하는 `REVIEW`로 식별합니다. `blocker`로 다음 라운드를 돌릴 때 같은 `task-id`에 새 `EXECUTED`/`REVIEW`를 추가합니다.
- 긴 설명은 `BATON-LOG.txt`에 직접 넣지 말고 `.baton/runs/`의 라운드 산출물로 분리합니다.

예시:

```text
2026-05-25T20:40:00 | qmxz | REQUEST  | Director | Fix typo in README
2026-05-25T20:41:00 | qmxz | RUN_DONE | Director | Fixed typo directly
2026-05-25T20:50:00 | abcd | REQUEST  | Director | Update protocol docs
2026-05-25T20:55:00 | abcd | PLANNED  | Planner | Plan written | .baton/runs/20260525-2055-docs-PLAN.md
2026-05-25T21:10:00 | abcd | EXECUTED | Executor | Changes submitted | .baton/runs/20260525-2055-docs-RUN-01.md
2026-05-25T21:15:00 | abcd | REVIEW   | Planner | No blockers | .baton/runs/20260525-2055-docs-REVIEW-01.md
2026-05-25T21:16:00 | abcd | CLOSE    | Director | Completed
```

사용자가 `CLOSE` 승인 전 결함을 알려준 경우 — 명백한 결함, 현재 런에 추가(같은 `task-id`):

```text
2026-05-26T10:20:00 | abcd | FEEDBACK | Director | User reported missing validation
2026-05-26T10:35:00 | abcd | EXECUTED | Executor | Fix submitted | .baton/runs/20260525-2055-docs-RUN-01.md
2026-05-26T10:40:00 | abcd | REVIEW   | Planner | No blockers | .baton/runs/20260525-2055-docs-REVIEW-01.md
```

피드백 후 새로운 런을 선택한 경우:

```text
2026-05-26T11:00:00 | abcd | FEEDBACK | Director | User requested scope change
2026-05-26T11:20:00 | abcd | EXECUTED | Executor | Changes submitted | .baton/runs/20260525-2055-docs-RUN-02.md
2026-05-26T11:25:00 | abcd | REVIEW   | Planner | No blockers | .baton/runs/20260525-2055-docs-REVIEW-02.md
```

`blocker`로 RUN-02가 필요한 경우(같은 `task-id`):

```text
2026-05-25T21:35:00 | abcd | EXECUTED | Executor | Changes submitted | .baton/runs/20260525-2055-docs-RUN-02.md
2026-05-25T21:40:00 | abcd | REVIEW   | Planner | No blockers | .baton/runs/20260525-2055-docs-REVIEW-02.md
```

## 6. 산출물

모든 라운드 산출물은 `.baton/runs/`에 작성하며, 작업당 하나의 안정된 `<YYYYMMDD>-<HHMM>-<slug>` 키를 씁니다. **이전 라운드를 덮어쓰지 않습니다.**

| 산출물 | 경로 | 작성자 | 의미 |
| --- | --- | --- | --- |
| Plan | `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-PLAN.md` | Planner | 계획과 성공 기준 |
| Submission | `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-RUN-<NN>.md` | Executor | 라운드 `<NN>`의 변경/검증/리스크 |
| Review | `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-REVIEW-<NN>.md` | Planner | 같은 라운드 발견 |
| Closure | `.baton/runs/<YYYYMMDD>-<HHMM>-<SLUG>-CLOSE.md` | Director | 사용자 승인 이후의 최종 종료 기록 |

- `<NN>`은 `01`부터 시작합니다.
- 이전 라운드를 덮어쓰지 않습니다. 예외: `FEEDBACK` 후 현재 런에 추가할 때, `CLOSE` 이벤트 승인 전이면 `RUN-<NN>.md` 갱신을 허용합니다.
- `<SLUG>`는 Director가 정한 소문자 kebab-case 작업 키를 씁니다.
- `<YYYYMMDD>`와 `<HHMM>`은 Director가 `REQUEST`를 기록할 때의 로컬 시스템 날짜·시분(24시간, 구분자 없음)을 씁니다. 예: `20260526-1430-diary-write`.
- `task-id`는 `BATON-LOG.txt` 이벤트 식별자이고, `<YYYYMMDD>-<HHMM>-<SLUG>`는 `.baton/runs/` 산출물 파일 키입니다. 같은 Relay 작업에서는 `task-id` 하나와 파일 키 하나를 함께 씁니다.
- 같은 작업의 모든 라운드 산출물은 같은 `<YYYYMMDD>-<HHMM>-<SLUG>` 키를 씁니다.
- 각 `CLOSE`는 완료된 작업이 `PLAN`과 어떻게 달라졌는지, 또는 달라지지 않았다면 `none`을 기록합니다.
- 산출물은 `.baton/templates/plan.md`, `run.md`, `review.md`, `close.md` 형식을 따릅니다.
- Executor는 긴 검증 전 `RUN-<NN>.md`를 checkpoint로 먼저 저장할 수 있습니다. TODO나 `<...>` placeholder가 남았거나 `Status: complete`가 아닌 RUN은 완료가 아니며, Director는 완료 검증 전 `EXECUTED`를 기록하지 않습니다.
- `PLAN`과 `REVIEW`에도 `Status` 줄이 있습니다. Planner는 보고 직전 마지막 쓰기로 정확히 `Status: complete`를 설정하며, Director는 이 줄로 완료를 감지합니다. `Status` 줄이 생기기 전에 작성된 과거 `PLAN`/`REVIEW`는 `baton lint`에서 그대로 통과합니다.
- Executor는 절대 `CLOSE`을 쓰지 않습니다.
- `REVIEW`는 사용자 결정을 위한 증거이지 승인이 아닙니다. Planner는 완료를 승인하거나 `CLOSE` 산출물을 작성하지 않습니다.
- **사용자가 명시적으로 승인한 뒤에만 Director가 `CLOSE` 산출물을 작성하고 `CLOSE` 이벤트를 기록하여 작업을 종료할 수 있습니다.**
- 어느 `REVIEW` 이후든 사용자의 `FEEDBACK`은 정상 파이프라인 단계이며, 완료된 작업의 예외적 되돌림으로 취급하지 않습니다.
- 각 `RUN`은 변경 파일, 변경 요약, 테스트/검증, 미해결 리스크를 기록합니다.
- `BATON-LOG.txt`는 `REQUEST`, `PLANNED`, `EXECUTED`, `REVIEW`, `FEEDBACK`, `CLOSE`, `RUN_DONE` 이벤트를 추가-전용으로 남기고 `path`로 산출물을 가리킵니다.
- 산출물 작성과 `BATON-LOG.txt` 이벤트 추가는 별개의 필수 작업입니다. Planner와 Executor는 산출물 완료와 제안 summary를 Director에게 통지할 뿐 `BATON-LOG.txt`를 쓰거나 기록 완료를 주장하지 않습니다. 각 단계는 Director가 해당 이벤트를 추가하고 확인한 뒤에만 완료된 것으로 봅니다.
- `.baton/bin/baton[.exe]`는 Director-owned 도구입니다. Director는 routine 흐름에서 `new-round`, `feedback`, `append`, `await`, `gate`, `status`, `prompt`를 사용합니다.
- 모든 `BATON-LOG.txt` 이벤트는 Director가 `baton append`로 추가합니다. 도구가 없거나 실패할 때만 직접 로그를 읽고 수동으로 복구합니다.
- Director는 다음 단계 위임 전에 `baton gate ...`로 직전 단계 이벤트를 확인합니다. 확인하지 못하면 다음 단계 위임을 중단하고 Director 소유의 로그 추가 또는 수정 작업을 완료합니다.
- 커밋 전이나 업데이트 후에는 `baton lint`로 Baton 상태를 검증합니다.

필수 게이트:

| 다음 단계 | 확인할 직전 이벤트 |
| --- | --- |
| Executor 위임 | `PLANNED` |
| Planner 검토 위임 | `EXECUTED` |
| 사용자 승인 요청 | `REVIEW` |
| 최종 `CLOSE` 이벤트 | 명시적 사용자 승인 |

## 7. 파이프라인

1. Director가 요청을 분류합니다.
2. 기록 제외 대상이면 응답만 하고 이벤트를 남기지 않습니다.
3. `Solo`이면 Director가 직접 처리하고 `REQUEST → RUN_DONE` 이벤트 흐름으로 작업을 닫습니다.
4. `Relay`이면 Director가 세션 Git 브랜치 전략을 적용한 뒤 `REQUEST`를 기록합니다.
5. Planner가 상단 `Director Brief`를 포함한 `PLAN`을 작성합니다.
6. Director는 기본적으로 `Director Brief`만 읽고 완전성을 확인한 뒤, 그 안의 `Executor Prompt`로 Executor에게 위임합니다.
7. Executor는 `PLAN`·성공 기준·범위에 따라 구현한 뒤 `RUN-01`을 쓰고, Director가 `EXECUTED`를 기록합니다.
8. Director가 `EXECUTED` 이벤트를 확인한 뒤 Planner에게 검토를 위임하고, Planner는 해당 `RUN` 경로를 받아 같은 번호의 `REVIEW`를 씁니다.
9. Director가 `REVIEW` 이벤트를 확인합니다. `blocker`가 없으면 사용자 결정 준비가 된 것이지 승인이 아닙니다. Director는 결과·검증, 존재하는 조치 대상 nit·리스크, 사용자가 직접 점검해야 할 케이스 3~5개, `REVIEW` 산출물 경로를 사용자에게 보고하고 승인을 요청합니다.
10. 사용자가 명시적으로 승인한 뒤에만 Director가 `CLOSE` 산출물을 작성하고 `CLOSE` 이벤트를 기록한 뒤 승인된 상태를 필요에 따라 커밋합니다. 전용 작업 브랜치를 사용했다면 기준 브랜치로 자동 병합합니다. 커밋 또는 병합에 문제가 있으면 강제하지 않고 blocker로 보고합니다.
11. 사용자가 승인 대신 피드백·결함을 알려주면 Director가 작업 브랜치에 `FEEDBACK`을 기록합니다. 명백한 결함이면 현재 런에 추가하고, 그렇지 않으면 **현재 런에 추가 / 새로운 런** 중 사용자 선택을 받습니다. 이후 Executor 작업부터 다시 진행합니다.
12. `CLOSE` 승인을 받은 Director는 해당 세션에서 발생한 착오, 해결 방법, 사용자 의견을 종합하여 `.baton/GUIDANCE.md` 수정안 또는 `.baton/lesson-learned/` 추가안을 사용자에게 제안합니다. 사용자가 수락한 항목만 기록합니다.
13. `blocker`가 있으면 Director가 다음 라운드를 Executor에게 위임하고, Executor가 `RUN-<NN>`을 쓰면 Director가 `EXECUTED`를 기록한 뒤 Planner가 다음 `REVIEW`를 씁니다. `REVIEW-03` 전까지 사용자 승인 없이 진행합니다.
14. `REVIEW-03`까지도 `blocker`가 남으면 Director는 상태를 보고하고 사용자에게 **재시도 / 계획 수정 / 부분 수락 / 중단** 중 선택을 요청합니다.

Relay 작업에서 사용자 개입이 필요한 경우는 `CLOSE` 최종 승인, `CLOSE` 승인 전 피드백·결함(`FEEDBACK`)과 `FEEDBACK` 후 현재 런·새 런 선택, `REVIEW-03` 이후에도 `blocker`가 남는 경우, Director가 사용자 결정이 필요하다고 판단한 경우뿐입니다. 전용 작업 브랜치를 쓴 경우 승인이 끝나면 병합은 자동으로 진행하며 별도 확인을 받지 않습니다.

`Solo` 작업은 사용자 완료 승인 없이 `RUN_DONE`으로 닫을 수 있습니다. 다만 장기 지침이나 재사용 가능한 교훈이 생겼다면 Director는 사용자에게 기록안을 제안하고, 사용자가 수락한 항목만 `GUIDANCE.md` 또는 `lesson-learned/`에 추가합니다.

## 8. 위임과 보고

### 컨텍스트 관리

Director, Planner, Executor는 한 작업 안에서 컨텍스트 교체 없이 연속 사용하는 것을 전제로 합니다.

Director는 컨텍스트가 불필요하게 커지지 않도록 위임 결과를 받을 때 산출물 **경로 + 최소 결정 정보**만 보관합니다.

- 한 줄 결과
- 한 줄 검증 상태
- 해당 시 `blocker` 건수/요약
- 잔존 리스크 또는 사용자 결정 요구

Planner는 각 `PLAN` 상단에 목표, 범위, 성공 기준, 리스크, 필수 확인, 최소 `Executor Prompt`를 담은 `Director Brief`를 작성합니다. Director는 Executor 위임 전 기본적으로 이 브리프만 읽고, 브리프가 누락·불완전·모순·고위험이거나 사용자 결정에 상세 검토가 필요한 경우에만 `PLAN` 전문을 읽습니다.

Planner/Executor는 가능하면 같은 컨텍스트를 유지하되, 다음 중 하나라도 발생하면 Director가 사용자에게 **교체 여부**를 물어야 합니다.

- 한 인스턴스에 후속 메시지가 5개 이상 누적
- 작업 주제가 명백히 바뀜
- 응답이 눈에 띄게 느려지거나 이전 컨텍스트를 혼동

### 위임 시 필수 필드

위임 프롬프트는 Director가 `baton prompt` 출력에 명시 요구사항만 덧붙여 만듭니다. Director는 세션 시작 때 확정한 역할별 모델과 reasoning effort를 호스트의 하위 에이전트 생성 옵션에 명시하며, 사용할 수 없는 모델을 임의로 대체하지 않습니다. 성공 기준·검증·리스크는 Planner가 `PLAN`에서 정의하고 Executor와 검토자는 이를 따릅니다. 하위 AI는 `baton`로 상태를 변경하거나 `BATON-LOG.txt`를 쓰지 않고, 완료와 제안 summary만 Director에게 알립니다.

### 보고 시 필수 필드

Planner가 Director에게 보고할 때는 다음만 간결히 포함합니다.

- `PLAN`/`REVIEW` 산출물 경로
- Director가 append할 이벤트명과 제안 summary
- 판단 결과
- `blocker` 수와 요약
- `nit` 요약
- 사용자 직접 점검 권장 케이스 3~5개
- 사용자 결정 필요 여부

Executor가 Director에게 보고할 때는 다음만 간결히 포함합니다.

- `RUN` 산출물 경로
- Director가 append할 이벤트명과 제안 summary
- 변경 요약
- 검증 결과
- 미해결 리스크
- 범위 밖으로 넘긴 사항

### 사용자에게 보여 주는 보고

사용자 대상 보고는 기본적으로 짧게 작성합니다. `Solo` 작업은 결과, 핵심
변경 범위, 검증만 1~3문장으로 알립니다. 생성·보존 파일 전체 목록, 프로토콜
진행 설명, 비어 있는 리스크/다음 단계 섹션은 사용자가 요청하거나 조치가
필요할 때만 포함합니다.

`Relay` 승인 요청은 결과, 검증 상태, 조치할 nit/리스크, 사용자 직접
점검 케이스 3~5개, `REVIEW` 경로만 우선 보여 줍니다. 상세 변경과 증거는
요청받지 않는 한 산출물에 둡니다.

## 9. 목표 파일 구조

Baton을 사용자 프로젝트에 도입했을 때의 `runs/` 중심 파일 구조입니다.

```text
project-root/
├── AGENTS.md                  # 기본 공통 지시문
├── CLAUDE.md                  # 선택, 동일 Baton 블록을 포함
└── .baton/
    ├── PROTOCOL.md
    ├── DIRECTOR.md
    ├── PLANNER.md
    ├── EXECUTOR.md
    ├── HOW-TO-UPDATE.md
    ├── VERSION
    ├── GUIDANCE.md             # 장기 지침/제약 누적
    ├── LESSON-LEARNED.md       # lesson-learned/ 실제 기록 인덱스
    ├── BATON-LOG.txt
    ├── bin/
    │   └── baton[.exe]     # 현재 플랫폼용 단일 Go CLI
    ├── lesson-learned/         # 완료 작업에서 얻은 해결 지식 기록
    │   └── .gitkeep
    ├── runs/                   # 라운드 산출물(PLAN/RUN/REVIEW/CLOSE)
    │   └── .gitkeep
    └── templates/
        ├── guidance.md
        ├── lesson-learned.md
        ├── plan.md
        ├── run.md
        ├── review.md
        └── close.md
```

## 10. 파일별 역할

| 파일 | 필수 여부 | 역할 |
|---|---:|---|
| `AGENTS.md` | 기본 | 모든 에이전트가 읽는 공통 작업 지침입니다. Claude 사용자가 `CLAUDE.md`만 유지하는 경우에는 생략할 수 있습니다. |
| `.baton/PROTOCOL.md` | 필수 | 모든 역할이 읽는 공통 규칙입니다. |
| `.baton/DIRECTOR.md` | 필수 | Director 전용 프로토콜입니다. |
| `.baton/PLANNER.md` | 필수 | Planner 전용 프로토콜입니다. |
| `.baton/EXECUTOR.md` | 필수 | Executor 전용 프로토콜입니다. |
| `.baton/HOW-TO-UPDATE.md` | 필수 | 이미 설치된 Baton을 업데이트할 때 따르는 절차입니다. |
| `.baton/VERSION` | 필수 | 설치 버전입니다. 업데이트 시 기본 upstream과 비교하는 기준으로 씁니다. |
| `.baton/GUIDANCE.md` | 누적 관리 | 세션을 넘어 유지할 사용자 지침, 제약, 금지사항을 담는 문서입니다. |
| `.baton/LESSON-LEARNED.md` | 누적 관리 | `.baton/lesson-learned/`에 저장된 실제 기록의 인덱스입니다. |
| `.baton/BATON-LOG.txt` | 필수 | 작업 이벤트 로그입니다 (추가 전용) |
| `.baton/bin/baton[.exe]` | 목표 필수 | 상태 변경, gate, 위임 프롬프트, 산출물·로그 검증, 지시 블록 병합, 업데이트를 처리하는 현재 플랫폼용 Go CLI입니다. |
| `.baton/lesson-learned/` | 누적 관리 | 완료된 작업에서 얻은 재사용 가능한 해결 지식이 쌓이는 디렉토리입니다. |
| `.baton/runs/` | 목표 필수 | `Relay` 작업의 라운드 산출물이 쌓이는 디렉토리입니다. |
| `.baton/templates/` | 목표 권장 | 산출물·누적 문서 작성 형식 예시입니다. |

## 11. 합류할 때 읽는 순서

새 세션을 시작하면 AI 에이전트는 현재 사용 도구가 읽는 지시 파일(`AGENTS.md`, `CLAUDE.md`, 또는 둘 다)을 우선 읽습니다. 두 파일의 `<baton-rules>...</baton-rules>` 블록은 동일하게 유지되므로 하나만 남아 있어도 됩니다. 여기서 Baton 안내를 확인하면 `.baton/PROTOCOL.md`와 자신의 역할별 프로토콜만 읽고, 그 규칙에 따라 프로젝트 맥락과 진행 중인 작업을 확인합니다.

Baton에 합류할 때의 읽기 순서는 다음과 같습니다.

```text
1. AGENTS.md 또는 CLAUDE.md에서 Baton 안내 확인
2. .baton/PROTOCOL.md
3. .baton/<ROLE>.md
4. .baton/GUIDANCE.md
4. .baton/LESSON-LEARNED.md 인덱스
5. 인덱스의 `Applies When` 또는 `Trigger / Symptom`이 현재 범위와 맞는 .baton/lesson-learned/ 기록
6. .baton/BATON-LOG.txt의 마지막 50줄
7. 진행 중인 라운드가 있으면 .baton/runs/의 최신 PLAN/RUN/REVIEW 읽기
```

같은 세션에서 연속 작업 중이라면 매 사용자 메시지마다 `BATON-LOG.txt`를 다시 읽지 않습니다. 다만 기록이 필요한 각 단계가 시작될 때 해당 역할은 `GUIDANCE.md`와 `LESSON-LEARNED.md` 인덱스를 다시 읽고, 자신의 현재 범위에 맞게 선택한 상세 기록만 읽습니다.

## 12. 기록해야 하는 작업

다음 작업은 `BATON-LOG.txt`에 결과를 남기는 것이 좋습니다.

- 코드 변경
- 문서 변경
- 설정 변경
- 테스트 실행
- 디버깅 시도
- 아키텍처 또는 설계 판단
- 이슈 상태 변경
- 다음 역할이나 후속 세션이 알아야 할 맥락 생성

`Relay`로 분류된 작업은 라운드 산출물(`PLAN`/`RUN`/`REVIEW`/`CLOSE`)이 본문 기록이고, `BATON-LOG.txt`는 그 산출물을 가리키는 이벤트 인덱스 역할을 합니다.

단순 질의응답, 짧은 설명, 브레인스토밍은 보통 기록하지 않습니다.
사용자가 맥락 보존을 명시적으로 요청한 경우에는 예외로 기록할 수 있습니다.

## 13. GUIDANCE 사용 기준

`GUIDANCE.md`는 기본 템플릿으로 복사됩니다.
부트스트랩 직후 억지로 채우는 문서가 아닙니다.

이 파일은 진행 상태, 현재 작업, 임시 계획, 다음 작업을 기록하는 곳이 아닙니다. 그런 정보는 `BATON-LOG.txt`와 라운드 산출물에 둡니다.

대신 Baton을 사용하는 동안 사용자가 지속적으로 주는 장기 지침을 누적합니다.

기록 대상:

- 오래 유지되는 프로젝트 맥락
- 사용자 선호와 상시 지시
- 기술·제품·운영 제약
- 하지 말아야 할 것
- 보안·개인정보 규칙
- 코드/테스트/브랜치/작업 관례

처음에는 비어 있거나 자리표시자가 남아 있어도 됩니다. 이후 장기 지침이 생길 때 갱신합니다.

갱신 시점:

- 사용자가 앞으로도 지켜야 할 지침을 말했을 때
- 사용자가 하지 말아야 할 일을 지정했을 때
- 보안, 개인정보, 운영상 제약이 추가되었을 때
- 코드 스타일, 테스트 방식, 브랜치 방식 같은 반복 적용 규칙이 생겼을 때

갱신하지 않는 경우:

- 현재 작업 진행 상황
- 임시 계획
- 디버깅 중간 메모
- 한 번만 적용되는 요청

기록 예시:

- "새 런타임 의존성은 사용자 확인 없이 추가하지 말 것"
- "공개 API는 하위 호환성을 유지할 것"
- "이 저장소에서는 pnpm을 사용할 것"

기록하지 않는 예시:

- "이 테스트를 고쳐줘"
- "먼저 다른 구현을 시도해봐"
- "현재 의심 원인은 race condition"

## 14. 부트스트랩 절차

대상 프로젝트에 Baton을 적용할 때의 실제 절차는 `BOOTSTRAP.md`를
기준으로 진행합니다. 아래는 `README.md`의 `runs/` 중심 목표 구조를
기준으로 한 핵심 절차입니다.

1. 부트스트랩 이후 작업을 맡을 `Director`, `Planner`, `Executor` 팀을 구성합니다.
2. `.baton/`가 이미 있으면 아무 파일도 바꾸지 않고 중단 후 보고합니다.
3. 대상 프로젝트의 `AGENTS.md`, `CLAUDE.md` 존재 여부를 확인합니다.
4. `AGENTS.md`가 없으면 `bootstrap/AGENTS.md`를 복사합니다.
5. `AGENTS.md`가 이미 있으면 기존 내용을 보존하고 `bootstrap/AGENTS.md`의 `<baton-rules>...</baton-rules>` 블록만 병합합니다.
6. Claude Code에서 실행 중이거나 `CLAUDE.md`가 이미 있으면 `CLAUDE.md`의 Baton 블록을 생성 또는 병합합니다.
7. `bootstrap/.baton/`를 복사합니다. `lesson-learned/`와 `runs/`는 빈 디렉토리(`.gitkeep`)로 생성합니다.
8. `BATON-LOG.txt`의 자리표시자 timestamp와 이벤트 줄을 현재 작업 정보에 맞게 바꿉니다.
9. `.baton/VERSION`에 설치 버전을 기록합니다.
10. `GUIDANCE.md`와 `LESSON-LEARNED.md`의 용도를 안내합니다.
11. 비밀정보가 `.baton/`에 들어가지 않았는지 확인합니다.
12. Git 저장소라면 `.baton/`가 커밋 대상인지 확인합니다.

## 15. 업데이트 절차

사용자가 "baton 최신화해줘", "업데이트해줘", "sync 해줘"처럼 요청하면 새 부트스트랩이 아니라 업데이트 절차를 수행합니다.

1. `.baton/VERSION`을 읽어 현재 설치 버전을 확인합니다.
2. 기본 upstream `https://github.com/grollcake/baton`의 최신 `main`을 임시 위치에 가져와 현재 프로젝트와 비교합니다.
3. `AGENTS.md`는 최신 `bootstrap/AGENTS.md`의 `<baton-rules>...</baton-rules>` 블록과 비교해 현재 파일의 Baton 블록만 교체하거나 보강합니다.
4. `CLAUDE.md`가 존재하면 최신 `bootstrap/CLAUDE.md`의 동일 블록과 비교해 Baton 블록만 교체하거나 보강합니다.
5. `.baton/bin/baton[.exe] update --upstream <repo>`로 dry-run을 확인하고, managed 파일의 로컬 수정을 덮어써도 될 때만 `--apply`를 붙입니다. Windows에서는 새 upstream 바이너리를 임시 경로에서 실행합니다.
6. `.baton/HOW-TO-UPDATE.md`, `.baton/PROTOCOL.md`, 역할별 프로토콜, `.baton/bin/baton[.exe]`, `.baton/templates/`는 로컬 수정이 없거나 안전히 구분될 때 최신 upstream으로 갱신합니다.
7. `.baton/GUIDANCE.md`, `.baton/lesson-learned/`, `.baton/BATON-LOG.txt`, `.baton/runs/`는 덮어쓰지 않습니다.
8. `.baton/LESSON-LEARNED.md`는 프로젝트별 기록 인덱스이므로 덮어쓰지 않습니다.
9. 업데이트가 성공하면 `.baton/VERSION`을 최신 upstream의 `VERSION` 값으로 갱신합니다.
10. Director는 `BATON-LOG.txt`에 `REQUEST → RUN_DONE`을 추가합니다. 메타 작업이라 기록을 생략하지 않습니다. 보통 `Solo`이며, `summary`에 이전·이후 `VERSION`을 포함합니다. 범위가 `Relay`에 해당하면 전용 작업 브랜치에서 `REQUEST → PLANNED → EXECUTED → REVIEW → CLOSE`을 기록하고, 승인 후 자동 병합합니다.

이전 버전의 `BATON-LOG.txt`가 `agent=`, `task=`, `TASK_BEGIN` 같은 형식을 사용하더라도 기존 줄은 수정하지 않습니다. 새 버전 적용 후 추가하는 이벤트부터 새 형식을 사용합니다.

`AGENTS.md`의 Baton 블록 안에 프로젝트 고유 지시가 섞여 있어 자동 분리가 어렵다면 파일을 바꾸지 않고 충돌로 보고합니다.

## 16. 중지와 제거

Baton은 설치와 업데이트만이 아니라 **멈추는 길**과 **나가는 길**을 가집니다. 되돌릴 수 있는 것과 없는 것이 다르므로 Director는 둘을 구분해 확인합니다.

| 사용자가 원하는 것 | 명령 | 되돌릴 수 있나 |
|---|---|---|
| 당분간 안 쓰고 싶다 | `<baton> pause` → `<baton> resume` | 예 |
| 이 저장소에서 완전히 뺀다 | `<baton> remove --apply` | 아니오 |

"Baton 그만 쓰자"는 요청은 대체로 중지를 뜻합니다. 모호하면 제거가 아니라 중지를 먼저 제안합니다.

### 중지

`pause`는 `AGENTS.md`·`CLAUDE.md`의 `<baton-rules>` 블록을 "중지됨" 문구로 바꿉니다. 지시 파일을 읽은 에이전트는 Baton이 설치되지 않은 것처럼 일합니다. `.baton/`의 기록과 설치 파일은 한 글자도 바뀌지 않으며, `resume`이 블록을 원래대로 정확히 되돌립니다.

중지 중에는 기록하거나 라운드를 진행하는 명령(`append`, `new-round`, `feedback`, `gate`, `prompt`, `await`, `merge-agent-block`, `update --apply`)이 거부하고 아무것도 쓰지 않습니다. 상태를 보는 명령(`status`, `lint`, `version`, `guide`, `check-artifact`, `models`, `update` dry run)은 동작하며 중지 상태임을 알립니다.

열린 작업이 있으면 `pause`는 거부합니다. 그래도 진행하려면 `--force`를 주고, 버려지는 작업이 기록에 남습니다.

**중지된 프로젝트는 업데이트 전에 `resume`이 필요합니다.** 업데이트가 규칙 블록을 다시 쓰면서 중지를 조용히 푸는 것을 막기 위해 `update`와 `merge-agent-block` 양쪽이 거부합니다. 순서는 `resume → update → pause`입니다.

### 제거

`remove`는 `--apply` 없이 실행하면 무엇을 지우고 무엇을 남길지만 출력합니다. 그 출력을 사용자에게 보여 주고 승인을 받은 뒤 실행합니다.

지우는 것은 Baton이 설치했고 **여전히 자기 것으로 알아보는** 바이트뿐입니다 — 관리 문서, 템플릿, `VERSION`, `bin/`, 그리고 지시 파일의 규칙 블록. Baton이 파일 전체를 썼던 지시 파일은 블록만 빠지는 것이 아니라 파일이 삭제되며, dry run이 어느 쪽인지 알려 줍니다.

남는 것은 프로젝트가 만든 기록입니다 — `BATON-LOG.txt`, `runs/`, `GUIDANCE.md`, `LESSON-LEARNED.md`, `lesson-learned/`, 그리고 Baton이 알아보지 못하는 모든 파일. `runs/`는 트리에 남은 코드가 왜 그렇게 되었는지 설명하는 유일한 문서인 경우가 많습니다.

기록까지 지우려면 `--purge`를 줍니다. 확인 대화 대신 **Git 전제조건**이 지킵니다: `.baton/` 아래 Git이 무시하지 않는 모든 경로가 추적되고 깨끗해야 통과합니다. 에이전트 파이프라인에서 확인 대화는 진행하려던 에이전트가 답하므로 안전장치가 되지 못하고, Git이 깨끗한지는 기계가 확인하며 거짓말할 수 없습니다.

거부되는 경우와 뜻은 다음과 같습니다.

| 거부 | 뜻 | 대처 |
|---|---|---|
| 열린 작업 N개 | 진행 중 작업이 기록만 남고 버려진다 | 작업을 닫거나 `--force` |
| 블록이 활성본·중지본 어느 쪽과도 다르다 | 마커 안에 사람이 쓴 글이 있다 | 그 글을 마커 밖으로 옮긴다. `--force`로도 통과하지 못한다 |
| `--purge`: 커밋되지 않았다 | 지우려는 기록이 Git 히스토리에 없다 | 먼저 커밋한다 |
| `--purge`: Git 저장소가 아니다 | 복구할 방법이 없다 | 직접 백업한 뒤 수동으로 지운다 |

마커 안 사용자 텍스트에 `--force`가 없는 것은 의도된 설계입니다. 그 바이트는 배포본에서 복원할 수 없습니다.

제거 후에는 `baton` 명령을 쓸 수 없고 `resume`으로 돌아올 수 없습니다. 다시 설치하려면 부트스트랩을 따르되, 1단계가 `.baton/`이 남아 있으면 중단하므로 사용자 확인을 거쳐 범위를 한정해 진행합니다.

### 절차 문서

설치된 프로젝트에서 실제 절차는 `.baton/HOW-TO-UPDATE.md` 한 곳에 있습니다. `<baton> guide remove`(또는 `pause`, `uninstall`, `lifecycle`)로 같은 문서를 바로 출력할 수 있습니다. 설치 전 독자를 위한 개요는 저장소 루트의 [`PAUSE.md`](PAUSE.md)와 [`UNINSTALL.md`](UNINSTALL.md)입니다.

## 17. `CLAUDE.md` 지시 파일

`CLAUDE.md`는 선택입니다. Claude Code에서 부트스트랩을 실행 중이거나 대상 프로젝트에 이미 `CLAUDE.md`가 있을 때만 추가하거나 병합합니다.

| 조건 | 처리 |
|---|---|
| Claude Code에서 실행 중이거나 `CLAUDE.md`가 이미 있음 | 없으면 생성하고, 있으면 기존 내용을 보존한 채 `bootstrap/CLAUDE.md`의 `<baton-rules>...</baton-rules>` 블록을 병합합니다. 이 블록은 `bootstrap/AGENTS.md`와 동일하게 유지합니다. |

`CLAUDE.md`가 없고 Claude Code에서 실행 중이지 않으면 새로 만들지 않습니다.

## 18. 보안 규칙

`.baton/`에는 다음 정보를 저장하지 않습니다.

- API 키
- 토큰
- 비밀번호
- 개인 키와 자격 증명
- 고객 데이터와 개인정보
- 민감한 내부 정보
- 운영 비밀

이 규칙은 `BATON-LOG.txt`, `GUIDANCE.md`, 라운드 산출물 모두에 적용됩니다.

## 19. 기타 규칙: Git 정책

Git 저장소에서는 `.baton/`를 커밋합니다.

Baton은 역할 및 세션 간 작업 인수인계를 위한 프로젝트 자산입니다. 릴레이 규칙, 프로젝트 지도, 작업 로그, 라운드 산출물은 다음 역할 또는 후속 세션이 같은 저장소에서 이어받을 수 있어야 합니다.

따라서 `.baton/`를 `.gitignore`에 추가하지 않습니다. 이미 무시 규칙이 있다면 제거합니다.

비밀정보는 커밋 여부와 무관하게 `.baton/`에 저장하지 않습니다.

## 20. 사용자에게 줄 수 있는 짧은 호출문

매번 긴 규칙을 붙일 필요는 없습니다.

```text
Baton 규칙에 따라 진행해.
```

더 명확히 지시하려면 다음처럼 말합니다.

```text
AGENTS.md 또는 CLAUDE.md와 .baton/PROTOCOL.md 기준으로 진행해.
새 세션이면 Baton의 읽기 순서를 먼저 따라줘.
```

영어 환경에서는 다음처럼 줄일 수 있습니다.

```text
Follow Baton. If this is a new or resumed session, follow the Baton read order before acting.
```

## 21. 배포 파일과 목표 템플릿 위치

현재 존재하는 배포 파일과 `runs/` 중심 목표 템플릿은 다음과 같습니다.

| 파일 | 용도 |
|---|---|
| `BOOTSTRAP.md` | 에이전트가 대상 프로젝트에 Baton을 설치할 때 따르는 절차 |
| `bootstrap/AGENTS.md` | 대상 프로젝트 루트에 둘 공통 지시문 |
| `bootstrap/CLAUDE.md` | Claude 환경에서 단독 유지할 수 있는 동일 Baton 블록 지시문 |
| `bootstrap/.baton/PROTOCOL.md` | Baton 공통 규칙 |
| `bootstrap/.baton/DIRECTOR.md` | Director 전용 프로토콜 |
| `bootstrap/.baton/PLANNER.md` | Planner 전용 프로토콜 |
| `bootstrap/.baton/EXECUTOR.md` | Executor 전용 프로토콜 |
| `bootstrap/.baton/HOW-TO-UPDATE.md` | 설치된 Baton 업데이트 절차 |
| `bootstrap/.baton/VERSION` | 설치 버전 템플릿 |
| `bootstrap/.baton/GUIDANCE.md` | 장기 지침/제약 누적 템플릿 |
| `bootstrap/.baton/LESSON-LEARNED.md` | `lesson-learned/` 실제 기록 인덱스 |
| `bootstrap/.baton/BATON-LOG.txt` | 초기 로그 템플릿 |
| `bootstrap/.baton/bin/<os>-<arch>/baton[.exe]` | 플랫폼별 Go CLI 바이너리 |
| `bootstrap/.baton/bin/SHA256SUMS` | 플랫폼별 바이너리 무결성 체크섬 |
| `bootstrap/.baton/lesson-learned/` | 완료 작업에서 얻은 해결 지식 기록 디렉토리 |
| `bootstrap/.baton/templates/guidance.md` | `GUIDANCE.md` 누적 지침 형식 |
| `bootstrap/.baton/templates/lesson-learned.md` | 해결 지식 기록 형식 |
| `bootstrap/.baton/templates/plan.md` | `PLAN` 산출물 형식 |
| `bootstrap/.baton/templates/run.md` | `RUN-NN` 산출물 형식 |
| `bootstrap/.baton/templates/review.md` | `REVIEW-NN` 산출물 형식 |
| `bootstrap/.baton/templates/close.md` | `CLOSE` 산출물 형식 |
| `cmd/baton/`, `internal/baton/` | Go CLI 진입점과 핵심 구현 |
| `cmd/build-release/` | 지원 플랫폼 바이너리와 체크섬 빌드 도구 |

## 22. 정리

Baton은 Director / Planner / Executor 에이전트 팀이 역할을 나누고, 이벤트 타임라인과 산출물을 기반으로 작업을 이어가기 위한 파일 기반 협업 규약입니다.

핵심 판단 기준은 세 개입니다.

```text
1) 이 요청은 기록 제외인가, Direct인가, Standard인가?
2) Standard라면 작업 브랜치에서 REQUEST → PLANNED → EXECUTED → REVIEW → CLOSE 흐름을 지키고 있는가?
3) 다음 역할 또는 후속 세션이 이벤트 타임라인과 산출물로 이어받을 수 있는가?
```

기록 제외 대상이면 응답만 하고 이벤트를 남기지 않습니다.
`Solo`이면 Director가 직접 처리하고 `REQUEST → RUN_DONE` 흐름이면 충분합니다.
