# Bootstrap

이 문서는 **AI 에이전트가 사용자 프로젝트에 Baton을 적용할 때 따라야 하는 절차**입니다.
사람이 직접 따라할 수도 있지만, 기본은 에이전트가 이 문서를 읽고 자동 수행합니다.

저장소 개요와 사용자용 한 줄 프롬프트는 [`README.md`](README.md)를 참조하세요.

---

## 사전 확인

부트스트랩을 시작하기 전, 사용자 프로젝트(이하 `<project>`)의 다음 상태를 확인합니다.

- 현재 OS와 아키텍처(`darwin|linux|windows`, `amd64|arm64`) 및 셸. 특정 셸은
  필수가 아니며, Git 연동 작업에는 `git`이 `PATH`에 있어야 합니다.
- `<project>/AGENTS.md` 존재 여부
- `<project>/.baton/` 디렉토리 존재 여부
- `<project>/.baton/VERSION` 존재 여부 (이미 설치된 경우 업데이트 기준)
- `<project>/CLAUDE.md` 존재 여부와 현재 실행 중인 에이전트 도구
- `<project>/README.md` 존재 여부 (없어도 진행 가능)

업데이트, 중지, 제거 절차는 이 문서가 아니라 설치된 프로젝트가 읽는
`.baton/HOW-TO-UPDATE.md`에 있습니다.

---

## 부트스트랩 절차

### 1. `.baton/` 디렉토리 처리

`<project>/.baton/` 존재 여부에 따라 분기합니다.

- **이미 있으면**: **부트스트랩을 중단**하고 사용자에게 보고합니다.
  - 이미 Baton이 적용된 프로젝트일 가능성이 큽니다.
  - 사용자가 "최신화", "업데이트", "sync"를 요청한 경우에는 새 설치가 아니라 아래 "업데이트 절차"를 따릅니다.
  - 사용자 확인 없이 기존 `AGENTS.md`, `BATON-LOG.txt`, `runs/`, `GUIDANCE.md`, `lesson-learned/`를 덮어쓰거나 머지하지 않습니다.
  - 사용자가 명시적으로 "재초기화" 또는 "특정 파일만 갱신"을 요청한 경우에만, 해당 범위로 한정해 진행합니다.
- **없으면**: 이후 단계로 진행합니다.

### 2. `AGENTS.md` 처리

`<project>/AGENTS.md` 존재 여부에 따라 분기합니다.

- **없으면**: `bootstrap/AGENTS.md`를 그대로 `<project>/AGENTS.md`로 복사합니다.
- **이미 있으면**: 덮어쓰지 말고 **머지**합니다.
  - 기존 내용 전체를 보존합니다.
  - 기존 파일에는 `bootstrap/AGENTS.md`의 `<baton-rules>...</baton-rules>` 블록만 추가합니다. 이 블록의 원본은 `bootstrap/AGENTS.md`이며, `BOOTSTRAP.md`에는 전문을 중복 유지하지 않습니다.
  - 동일한 Baton 블록이나 포인터가 이미 있으면 중복 추가하지 않고 누락된 문장만 보강합니다.
  - 기존 프로젝트의 규칙·역할·지시문을 임의로 삭제하거나 재작성하지 않습니다.

### 3. `CLAUDE.md` 처리 (선택)

현재 Claude Code에서 실행 중이거나 대상 프로젝트에 `CLAUDE.md`가 이미
있는 경우에만 `CLAUDE.md`를 생성하거나 머지합니다.

| 트리거 | 동작 | 원본 |
|---|---|---|
| Claude Code에서 실행 중이거나 `<project>/CLAUDE.md` 존재 | 없으면 생성, 있으면 기존 내용 보존 + Baton 블록 머지 | `bootstrap/CLAUDE.md` |

머지 규칙:

- `CLAUDE.md`에는 `bootstrap/CLAUDE.md`의 `<baton-rules>...</baton-rules>` 블록을 추가하거나 보강합니다. 이 블록은 `bootstrap/AGENTS.md`의 동명 블록과 동일하게 유지합니다.
- 따라서 Claude 사용자는 설치 후 `AGENTS.md`를 제거하고 `CLAUDE.md`만 유지해도 같은 Baton 규칙을 유지할 수 있습니다.
- 동일한 Baton 블록이 이미 있으면 중복 추가하지 않습니다.
- `CLAUDE.md`가 없고 현재 Claude Code에서 실행 중이지 않으면 새로 생성하지 않습니다.

### 4. `.baton/` 디렉토리와 바이너리 복사

1. `bootstrap/.baton/`에서 `bin/`을 제외한 내용을
   `<project>/.baton/`로 복사합니다.
2. 현재 플랫폼 키를 `<os>-<arch>`로 정합니다. 지원 키는
   `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`,
   `windows-amd64`, `windows-arm64`입니다.
3. macOS/Linux에서는
   `bootstrap/.baton/bin/<os>-<arch>/baton`를
   `<project>/.baton/bin/baton`로 복사하고 실행 권한을 줍니다.
4. Windows에서는
   `bootstrap/.baton/bin/windows-<arch>/baton.exe`를
   `<project>/.baton/bin/baton.exe`로 복사합니다.
5. `bootstrap/.baton/bin/SHA256SUMS`에서 선택한 플랫폼 항목을 확인해
   복사된 바이너리의 SHA-256을 검증하고, `SHA256SUMS`도 대상 `bin/`에
   복사합니다.
6. 대상 프로젝트에는 선택한 바이너리 하나만 둡니다. Go 런타임이나 소스는
   복사하지 않습니다.

### 5. 초기 `BATON-LOG.txt` 기록

복사된 `<project>/.baton/BATON-LOG.txt`에는 부트스트랩 작업을 기록하기 위한 두 줄이 들어 있습니다. 이 두 줄의 placeholder를 실제 값으로 바꿉니다.

1. 현재 로컬 시스템 시간을 `YYYY-MM-DDTHH:MM:SS` 형식으로 적습니다.
2. 무작위 소문자 영문 4글자로 `task-id` 하나를 만듭니다.
3. `REQUEST` 줄과 `RUN_DONE` 줄에 같은 `task-id`를 넣습니다.
4. 두 줄의 timestamp를 실제 부트스트랩 시작/완료 시각으로 바꿉니다. 같은 시각을 써도 됩니다.

예시:

```text
2026-04-28T19:45:00 | abcd | REQUEST  | Director | Bootstrap Baton
2026-04-28T19:46:00 | abcd | RUN_DONE | Director | Baton initialized
```

### 6. `GUIDANCE.md`와 `LESSON-LEARNED.md` 안내

`GUIDANCE.md`는 기본 운영 관례가 포함된 템플릿으로 복사됩니다. 부트스트랩 직후 프로젝트별 지침을 억지로 추가하지 않습니다.

`LESSON-LEARNED.md`는 `.baton/lesson-learned/`에 저장된 실제 기록의 검색 인덱스로 복사됩니다. 인덱스는 `Applies When`과 `Trigger / Symptom`으로 단계별 담당 역할이 관련 상세 기록만 선택해 읽도록 합니다. 기록은 작업 완료 승인 이후 사용자가 수락한 항목만 추가하고, 수락된 기록 파일을 이 인덱스에 함께 등록합니다.

### 7. Git 정책 확인

`<project>`가 Git 저장소라면 `.baton/`는 커밋 대상입니다. 이미 `.gitignore`에 `.baton/`가 있다면 제거해야 합니다.

### 8. 완료 전 프로토콜 확인

부트스트랩 완료를 보고하기 전에 에이전트는 반드시 대상 프로젝트의 `<project>/.baton/PROTOCOL.md`를 읽어야 합니다. 이 확인을 마치기 전에는 부트스트랩 완료를 보고하지 않습니다.

---

## 완료 보고 형식

부트스트랩이 끝나면 사용자에게 기본적으로 결과만 짧게 보고합니다. 사용자가
상세 내역을 요청했거나 충돌, 수동 조치, 검증 실패가 있을 때만 파일별 목록과
추가 설명을 펼칩니다.

```text
Baton v<x.x> 부트스트랩을 완료했습니다. 이제 이 프로젝트는 Baton 규칙을 따릅니다.
```

---

## 업데이트 절차

사용자가 이미 Baton이 적용된 프로젝트에서 "baton 최신화해줘", "업데이트해줘", "sync 해줘"처럼 요청하면 새 부트스트랩을 하지 않고 다음 절차를 따릅니다.

### 1. 설치 버전 읽기

`<project>/.baton/VERSION`을 읽습니다.

- 없으면 설치 버전을 알 수 없으므로 중단하고 사용자에게 보고합니다.
- 있으면 그 값을 현재 설치 버전으로 보고, 기본 upstream `https://github.com/grollcake/baton`의 `main` 브랜치와 비교합니다.

### 2. upstream 확보와 비교

기본 upstream의 최신 `main`을 임시 위치에 가져와 현재 프로젝트와 비교합니다.

- 현재 프로젝트 파일과 최신 upstream을 직접 비교하되, 프로젝트별 내용은 보수적으로 보존합니다.

### 3. 파일별 업데이트 정책

| 대상 | 업데이트 방식 |
|---|---|
| `AGENTS.md` | 최신 `bootstrap/AGENTS.md`의 `<baton-rules>...</baton-rules>` 블록과 비교해 현재 파일의 Baton 블록만 교체 또는 보강 |
| `.baton/HOW-TO-UPDATE.md` | 로컬 수정이 없거나 안전히 구분되면 최신 upstream으로 갱신 |
| `.baton/PROTOCOL.md` | 로컬 수정이 없거나 안전히 구분되면 최신 upstream으로 갱신 |
| `.baton/DIRECTOR.md` | 로컬 수정이 없거나 안전히 구분되면 최신 upstream으로 갱신 |
| `.baton/PLANNER.md` | 로컬 수정이 없거나 안전히 구분되면 최신 upstream으로 갱신 |
| `.baton/EXECUTOR.md` | 로컬 수정이 없거나 안전히 구분되면 최신 upstream으로 갱신 |
| `.baton/bin/baton[.exe]` | 현재 OS/아키텍처에 맞는 최신 upstream Go 바이너리로 교체 |
| `.baton/templates/` | 템플릿 파일은 최신 upstream과 비교해 갱신 |
| `.baton/VERSION` | 성공 후 최신 upstream의 `VERSION` 값으로 갱신 |
| `CLAUDE.md` | 존재하는 경우 최신 `bootstrap/CLAUDE.md`의 `<baton-rules>...</baton-rules>` 블록과 비교해 현재 파일의 Baton 블록만 교체 또는 보강 |
| `.baton/GUIDANCE.md` | 덮어쓰지 않음 |
| `.baton/LESSON-LEARNED.md` | 프로젝트별 기록 인덱스이므로 덮어쓰지 않음 |
| `.baton/BATON-LOG.txt` | 기존 줄을 수정하지 않음. 이전 버전의 다른 형식도 보존하고, 새 이벤트부터 최신 `REQUEST`, `PLANNED`, `EXECUTED`, `REVIEW`, `FEEDBACK`, `CLOSE`, `RUN_DONE` 형식을 사용 |
| `.baton/runs/` | 덮어쓰지 않음 |
| `.baton/lesson-learned/` | 덮어쓰지 않음 |

### 4. 바이너리 업데이트 규칙

- macOS/Linux는 `.baton/bin/baton`, Windows는
  `.baton/bin/baton.exe` 하나만 유지합니다.
- 새 upstream의 `bootstrap/.baton/bin/<os>-<arch>/`에서 현재 플랫폼과
  일치하는 바이너리를 선택합니다.
- Windows에서 실행 중인 설치 바이너리는 자신을 교체할 수 없으므로, 새
  upstream 바이너리를 임시 경로에서 실행하여 `update --apply`를 수행합니다.
- `0.10.x` 이하처럼 바이너리가 없는 기존 설치도 새 upstream의 플랫폼
  바이너리를 대상 프로젝트 디렉토리에서 실행하여 `update --apply`합니다.
- 성공한 업데이트는 기존 `.baton/scripts/`가 있으면 제거합니다.

### 5. `AGENTS.md` 머지 규칙

`AGENTS.md`는 프로젝트별 지시가 가장 많이 섞이는 파일이므로 다음 원칙을 따릅니다.

- 기존 프로젝트 규칙, 역할, 금지사항, 빌드/테스트 지시를 삭제하지 않습니다.
- `<baton-rules>...</baton-rules>` 블록이 있으면 그 블록만 최신 `bootstrap/AGENTS.md`의 동명 블록과 비교합니다.
- Baton 포인터만 있고 블록이 없으면 최신 블록을 추가하되 중복 문장은 제거합니다.
- Baton 블록 안에 프로젝트 고유 지시가 섞여 있어 자동 분리가 어렵다면 파일을 바꾸지 않고 충돌로 보고합니다.

### 6. `BATON-LOG.txt` 기록

업데이트 완료 후 Director는 `BATON-LOG.txt`에 `REQUEST → RUN_DONE`을 추가합니다. 메타 작업이라 기록을 생략하지 않습니다. `summary`에 이전·이후 `VERSION`을 포함합니다.

예시:

```text
2026-05-26T00:10:00 | kfnp | REQUEST  | Director | Update Baton 0.33 -> 0.34
2026-05-26T00:12:00 | kfnp | RUN_DONE | Director | Baton updated to 0.34
```

### 7. 완료 보고

업데이트가 끝나면 기본적으로 버전 변화, 갱신 범주, 수동 확인 필요 여부만
짧게 보고합니다. 변경 파일 전체 목록과 보존 파일 목록은 요청받거나 충돌이
있을 때만 보고합니다.

```text
Baton 업데이트 완료. v<이전> -> v<이후>; 갱신: <범주>. 수동 확인: <없음 또는 대상>.
```

---

## 중단·실패 보고 형식

`.baton/`가 이미 있어 진행을 중단했거나, 머지 중 충돌이 발생한 경우 다음과 같이 보고합니다.

```text
Baton 부트스트랩을 중단했습니다.

이유:
- <project>/.baton/ 가 이미 존재
- 기존 BATON-LOG.txt 마지막 이벤트: <timestamp> | <event_type> | ...

확인이 필요한 사항:
- 재초기화를 원하시면 .baton/를 백업 후 알려주세요.
- 일부 파일만 갱신하려면 어떤 파일을 갱신할지 지정해주세요.
```

---

## 참고

- 사양 전체: [`bootstrap/.baton/PROTOCOL.md`](bootstrap/.baton/PROTOCOL.md)
- 한국어 가이드: [`PROTOCOL-GUIDE.md`](PROTOCOL-GUIDE.md)
