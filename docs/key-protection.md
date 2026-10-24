# KEDP 키 보호 5겹 가이드

2026-10-23 · KEDP001 (`aidev-KEDP001`)

## 목적

ECOS, KOSIS, 공공데이터포털의 API 키가 Git, GitHub, 로그로 새어 나가지 않게 다섯 겹으로 막습니다.

- **키가 주소 안에 들어갑니다.** 주소가 적히는 곳(로그, 에러 메시지, 샘플 파일)에 키가 함께 남기 쉽습니다.
- **Git은 한 번 올라간 것을 기억합니다.** 파일을 지워도 과거 commit 기록에 남습니다.
- **키는 Sun&Moon 이름의 출입증입니다.** 새면 남이 쓰거나, 사용량이 넘쳐 사이트가 키를 막습니다.
- **사람도 Agent도 실수합니다.** 한 겹이 놓친 것을 다음 겹이 잡습니다.

---

## 1. 5겹 한눈에

```text
 키가 흘러가는 길                          막는 겹
 ──────────────────────────────────────────────────────────────
 키를 보관
   └─ .env 파일이 Git에 담김 ───────────▶ 1겹  .gitignore 로 제외
 Go 프로그램 실행
   └─ 로그·에러·샘플에 주소 기록 ────────▶ 2겹  키를 *** 로 가리기
 저장 (git commit)
   └─ 실제 키 값이 파일에 섞임 ──────────▶ 3겹  실제 키 값 대조    ┐ 내 컴퓨터
   └─ 키처럼 생긴 값이 섞임 ─────────────▶ 4겹  gitleaks 검사      ┘ (hook)
 올리기 (git push) → PR
   └─ hook 없이 올라온 키 ───────────────▶ 5겹  GitHub 자동 검사
 원본(main)에 반영
```

| 겹 | 동작 위치·시점 | 막는 것 | 못 막는 것 → 대신 막는 겹 | 상태 |
| --- | --- | --- | --- | --- |
| 1 | Git, `git add` 할 때 | `.env` 파일이 담기는 것 | 강제로 담기, 다른 파일에 섞인 키 값 → 3겹 | 완료 |
| 2 | Go 프로그램, 실행 중 | 로그·에러·샘플에 키가 적히는 것 | 가리기를 거치지 않은 출력 → 3·4겹, 약속 | Go 코드를 처음 쓸 때 |
| 3 | 내 컴퓨터, commit 직전 | 실제 키 값이 파일에 섞인 것 | `.env`에 없는 값, 형태가 바뀐 값 → 4겹 | 완료 (PR #1) |
| 4 | 내 컴퓨터, commit 직전 | 키처럼 생긴 값 | hook 미설치, 검사 건너뛰기 → 5겹 | 완료 (PR #1) |
| 5 | GitHub, PR을 올릴 때 | hook 없이 올라온 키를 **발견** | 이미 올라간 사실 → 11장 사고 대응 | 완료 (PR #2) |

**만드는 순서는 번호와 다릅니다.**

```text
1겹 → 3·4겹 → 5겹 → 실제 키 넣기 → 2겹
```

키를 넣기 **전에** 막는 장치(1·3·4·5겹)를 먼저 갖춥니다. 2겹은 키를 쓰는 Go 코드 안에 들어가는 기능이라, Go 코드를 처음 쓸 때 가장 먼저 만듭니다.

---

## 2. 공통: 준비, 등록 흐름, 막히면

### 2-1. 준비 확인

```bash
cd /srv/projects/aidev-kedp001
gitleaks version              # 8.30.1 (4겹이 씀, 없으면 hook이 commit을 막음)
gh auth status                # Logged in to github.com account schware
git branch --show-current     # 새 작업은 항상 main에서 시작
```

### 2-2. 등록 흐름 (branch → PR)

1겹(저장소 첫 commit)을 뺀 모든 겹은 이 흐름으로 등록합니다. 각 겹의 따라하기에는 이 흐름에 값을 채운 명령이 있습니다.

| 순서 | 명령 | 뜻 |
| --- | --- | --- |
| 1 | `git switch main` | 원본으로 이동 |
| 2 | `git pull` | GitHub의 최신 원본을 받음. 빠뜨리면 오래된 원본에서 작업하게 됨 |
| 3 | `git switch -c <branch>` | 사본을 만들고 이동. 원본은 직접 고치지 않음. 이름은 `담당/작업내용` |
| 4 | (파일 만들기) | |
| 5 | `git status` | `Untracked files`에 새 파일이 보이는지 확인 |
| 6 | `git add <파일>` | 저장할 파일을 담음 |
| 7 | `git status` | `Changes to be committed`에 보이는지 확인. **commit은 담은 파일만 기록함** |
| 8 | `git commit -m "<메시지>"` | 저장. 이때 3·4겹 hook이 자동으로 돎 |
| 9 | `git push -u origin <branch>` | GitHub에 올림. `-u`는 사본끼리 연결해서 다음부터 `git push`만 쳐도 되게 함 |
| 10 | `gh pr create ...` | 결재 요청(PR)을 만듦 |
| 11 | `gh pr checks <branch> --watch` | 5겹 검사 결과를 기다림 (5겹 등록 이후) |
| 12 | `gh pr merge <branch> --squash --delete-branch` | 원본에 반영. commit들을 하나로 묶고, 다 쓴 사본을 지움 |
| 13 | `git switch main` → `git pull` | 반영된 원본을 Linux로 받음. **가장 자주 잊는 단계** |

### 2-3. 막히면 (공통)

| 화면에 나온 말 | 원인 | 해결 |
| --- | --- | --- |
| `nothing added to commit but untracked files present` | 담기(add)를 안 함 | `git add <파일>` 후 다시 commit |
| `No commits between main and ...` | 사본에 commit이 없음 | 사본으로 돌아가 add, commit, push 후 다시 PR |
| `no pull requests found for branch` | PR을 만들지 않고 merge함 | `gh pr create` 먼저 |
| `fatal: a branch named ... already exists` | 같은 이름의 사본이 있음 | `-c` 없이 `git switch <branch>` |
| `fatal: not a git repository` | 프로젝트 폴더가 아님 | `cd /srv/projects/aidev-kedp001` |

---

## 3. 1겹: `.env` 파일을 Git에서 뺀다

### 왜 필요한가

- **키는 코드와 따로 보관합니다.** 코드는 팀원 모두가 보고 GitHub에 올리는 것이고, 키는 사람마다 다르고 남에게 보여주면 안 되는 것입니다. 그래서 키는 `.env` 파일 **한 곳에** 모으고, 프로그램은 실행할 때 이 파일에서 읽습니다.
- **그런데 Git은 폴더 안의 모든 파일을 담습니다.** `git add .` 한 번이면 `.env`도 담기고, commit되면 기록에 영원히 남습니다.
- **그래서 Git에게 "이 파일은 없는 것처럼 봐라"고 알려 줍니다.** 그 목록이 `.gitignore`입니다. 실수할 기회 자체를 없애는 가장 기본 겹입니다.
- **`.env.example`도 함께 둡니다.** 키 값 없이 **이름만** 적은 견본이라, 팀원이 어떤 키가 필요한지 알고 복사해서 채웁니다.

### 따라하기

**① `.gitignore` 만들기**

```bash
cat > .gitignore <<'EOF'
# Secrets: never commit
.env
.env.*
!.env.example

# Build outputs
target/
node_modules/
dist/
EOF
```

> `.env`, `.env.local`, `.env.backup` 같은 이름은 모두 제외하고, 견본 `.env.example`만 예외(`!`)로 올립니다. 아래 세 줄은 크고 매번 바뀌는 빌드 결과물 폴더입니다.

**② `.env.example` 만들기**

```bash
cat > .env.example <<'EOF'
ECOS_API_KEY=
KOSIS_API_KEY=
DATA_GO_KR_API_KEY=
EOF
```

**③ 확인**

```bash
git check-ignore -v .env
```

결과: `.gitignore:2:.env	.env` → 2번째 줄 규칙으로 `.env`가 제외됨

**④ 등록 (저장소 처음 만들 때만 main에 직접)**

```bash
git add .gitignore .env.example
git commit -m "Initialize repository with .gitignore and .env.example"
gh repo create aidev-KEDP001 --private --source=. --remote=origin --push
```

### 막히면

| 증상 | 해결 |
| --- | --- |
| `git check-ignore`가 아무것도 출력하지 않음 | `cat .gitignore`로 내용과 위치 확인 |
| `.env.example`도 제외됨 | `!.env.example` 줄이 있는지 확인 |

---

## 4. 2겹: Go 코드에서 키를 `***`로 가린다

### 왜 필요한가

나머지 겹은 모두 **Git에 올라가는 것**만 막습니다. 하지만 키는 프로그램이 실행되는 동안 Git 밖에서도 샙니다. 예를 들어 Go는 요청이 실패하면 에러 메시지에 주소 전체를 담습니다.

```text
Get "https://ecos.bok.or.kr/api/StatisticSearch/ABCD1234/json/...": timeout
```

이 메시지는 서버 로그 파일, 터미널 화면(캡처해서 팀원·Jira에 공유), Claude Code에게 붙여 넣는 대화, 샘플 파일(`samples/`)로 퍼집니다. 이 길들은 Git을 거치지 않아서 다른 겹이 막지 못합니다.

### 장점

1. **출발점에서 막습니다.** 키가 기록에 처음부터 들어가지 않아, 뒤의 겹에 걸릴 일 자체가 줄어듭니다.
2. **Git 밖의 유출도 막습니다.** 로그 공유, 화면 캡처, 에러를 Claude Code에게 보여주는 것이 안전해집니다.
3. **한 곳만 고치면 됩니다.** KEDP에서 외부 API를 부르는 것은 **Go(`b-go-end/`)뿐**입니다. TypeScript와 Rust는 직접 부르지 않습니다.
4. **샘플 규칙을 자동으로 지킵니다.** `samples/README.md`의 "API 키와 인증정보는 제거한다"를 사람이 손으로 하지 않아도 됩니다.

### 어떻게 동작하나

키를 주소에 붙이는 일과 주소를 기록하는 일을 **Go 코드 한 곳에서만** 하고, 기록할 때는 항상 가린 주소를 씁니다.

```text
실제 요청:  https://ecos.bok.or.kr/api/StatisticSearch/ABCD1234/json/...
기록할 때:  https://ecos.bok.or.kr/api/StatisticSearch/***/json/...
```

### 따라하기

아직 Go 코드가 없어서 지금은 할 것이 없습니다. Go로 ECOS를 처음 호출하는 작업(branch 예: `go/ecos-sample`)에서 **가리기 기능을 가장 먼저** 만들고, 이 장에 따라하기를 추가합니다.

---

## 5. 3겹: commit 직전, 실제 키 값이 섞였는지 검사한다

### 어떤 경우의 이슈인가

1겹은 `.env` **파일**만 막습니다. 하지만 키 **값**은 복사, 붙여 넣기로 다른 파일에 쉽게 섞입니다.

| 경우 | 예시 |
| --- | --- |
| 빨리 시험하려고 코드에 키를 직접 넣음 | `url := "https://ecos.bok.or.kr/api/StatisticSearch/ABCD1234/..."` |
| 샘플을 저장하면서 요청 주소를 메모로 남김 | `samples/ecos/base-rate.json`의 `"request_url"` |
| 테스트 코드에 실제 키를 넣음 | `const testKey = "ABCD1234..."` |
| 문서에 예시 주소를 쓰면서 실제 키를 넣음 | `specification/10.indicators/README.md` |
| Agent가 `.env`를 읽고 그 값을 다른 파일에 씀 | 설정 파일, 테스트 데이터 |
| `.env`를 강제로, 또는 다른 이름으로 담음 | `git add -f .env`, `.env.backup` |

### 어떻게 검사하나

1. `.env`에 있는 실제 키 값을 하나씩 꺼냅니다.
2. 이번에 commit하려는 **새로 추가된 줄**에 그 값이 글자 그대로 있는지 찾습니다.
3. 있으면 commit을 멈춥니다. 메시지에는 값 대신 **이름**만 표시합니다.

```text
[pre-commit] Blocked: the value of ECOS_API_KEY from .env is in the staged changes.
```

### 왜 commit 직전인가

commit 직전에는 아직 **내 컴퓨터 안에만** 있습니다. 파일만 고치면 끝납니다. GitHub에 올라간 뒤에 발견하면 키를 바꿔야 합니다(11장).

설치는 4겹과 한 파일이라 7장에서 함께 합니다.

---

## 6. 4겹: commit 직전, 키처럼 생긴 값을 검사한다

### 무엇을 검사하나

gitleaks가 commit하려는 변경 내용에서 **비밀값처럼 생긴 것**을 찾습니다. gitleaks에는 수백 개의 규칙이 있습니다.

- 알려진 서비스의 키 형식 (예: GitHub 토큰은 `ghp_`로 시작, AWS 키는 `AKIA`로 시작)
- `api_key = "..."`, `password: ...`처럼 이름 옆에 붙은 값
- 길고 무작위한 영문·숫자 조합

### 3겹과 4겹의 차이는 어떻게 생기나

**검사 기준이 다르기 때문입니다.** 3겹은 우리 키(`.env`)라는 **정답지와 대조**하고, 4겹은 정답지 없이 **생김새로 의심**합니다.

| | 3겹: 실제 키 값 대조 | 4겹: gitleaks |
| --- | --- | --- |
| 기준 | `.env`에 있는 우리 키 | 비밀값의 일반적인 모양 |
| 비유 | 지명수배 사진과 얼굴 대조 | 수상한 행동으로 의심 |
| 확실함 | 걸리면 100% 우리 키 | 키가 아닌 것도 가끔 걸림 (예외 등록으로 해결) |
| 잘 잡는 것 | 주소 중간에 그냥 들어간 ECOS·KOSIS 키 | `.env`에 없는 비밀값 (다른 팀원의 키, GitHub 토큰, DB 비밀번호) |
| 못 잡는 것 | `.env`에 없는 값, 형태가 바뀐 값, 8자 미만 값 | 정해진 형식이 없는 평범한 모양의 키 |

**둘 다 두는 이유**

- ECOS 키처럼 정해진 형식이 없는 키가 주소 중간에 들어가면, gitleaks는 키인지 알아보지 못할 수 있습니다. → 3겹이 잡습니다.
- 팀원 A의 키가 섞인 파일을 내가 commit하면, 내 `.env`에는 A의 키가 없어서 3겹은 모릅니다. → 4겹이 모양으로 잡습니다.

---

## 7. 3·4겹 설치: commit 전 검사 hook

3겹과 4겹은 `.githooks/pre-commit` **파일 하나**에 들어 있습니다. Git은 commit 직전에 이 파일을 자동으로 실행하고, 파일이 실패로 끝나면 commit을 취소합니다.

### ① 사본 만들기

```bash
git switch main
git pull
git switch -c common/key-guard
```

> 검사 도구는 특정 언어 담당이 아니라서 `common`을 씁니다.

### ② hook 파일 만들기

```bash
mkdir -p .githooks
```

```bash
cat > .githooks/pre-commit <<'EOF'
#!/usr/bin/env bash
# Pre-commit guard: blocks commits that would expose API keys.
set -euo pipefail

# 1) Never commit .env files (except .env.example)
if git diff --cached --name-only --diff-filter=ACM \
  | grep -E '(^|/)\.env(\.[^/]*)?$' | grep -vE '(^|/)\.env\.example$' | grep -q .; then
  echo "[pre-commit] Blocked: a .env file is staged." >&2
  exit 1
fi

# 2) Compare staged changes against the real key values in .env
if [ -f .env ]; then
  added=$(git diff --cached -U0 --no-color | grep '^+' | grep -v '^+++' || true)
  while IFS='=' read -r name value; do
    case "$name" in ''|\#*) continue ;; esac
    value="${value%\"}"; value="${value#\"}"
    [ "${#value}" -lt 8 ] && continue
    if printf '%s' "$added" | grep -qF -- "$value"; then
      echo "[pre-commit] Blocked: the value of $name from .env is in the staged changes." >&2
      exit 1
    fi
  done < .env
fi

# 3) Pattern scan with gitleaks
if ! command -v gitleaks >/dev/null 2>&1; then
  echo "[pre-commit] Blocked: gitleaks is not installed." >&2
  exit 1
fi
gitleaks git --pre-commit --staged --redact --verbose
EOF
```

**hook 파일 해설**

| 부분 | 뜻 |
| --- | --- |
| `#!/usr/bin/env bash` | 이 파일을 bash로 실행하라는 표시 |
| `set -euo pipefail` | 중간에 에러가 나면 바로 멈춤. 검사가 반쯤 돌고 통과되는 일을 막음 |
| `exit 1` | 실패로 끝냄. Git은 hook이 실패하면 commit을 취소함 |
| **`# 1)` 블록** | **1겹 보완**: 담긴 파일 중 `.env`, `.env.*`가 있으면 차단 (`.env.example`은 허용) |
| `git diff --cached --name-only` | 이번 commit에 담긴 파일 이름 목록 |
| **`# 2)` 블록** | **3겹**: `.env`가 있을 때만 실행 |
| `git diff --cached -U0 ... grep '^+'` | 이번 commit에서 **새로 추가되는 줄**만 꺼냄 |
| `while IFS='=' read -r name value` | `.env`를 한 줄씩 `이름=값`으로 나눠 읽음 |
| `case ... continue` | 빈 줄과 `#` 주석 줄은 건너뜀 |
| `value="${value%\"}"; value="${value#\"}"` | 값 앞뒤의 따옴표를 떼어 냄 |
| `[ "${#value}" -lt 8 ] && continue` | 8자 미만 값은 건너뜀. 빈 값이나 짧은 설정값이 엉뚱하게 걸리는 것을 막음 |
| `grep -qF -- "$value"` | 값을 **글자 그대로** 찾음. `-F`는 특수문자도 글자로 취급 |
| `echo "... $name ..."` | 차단 메시지에 값 대신 **이름**만 표시 |
| **`# 3)` 블록** | **4겹**: gitleaks가 없으면 차단, 있으면 검사 |
| `--pre-commit --staged` | 이번 commit에 담긴 내용만 검사 |
| `--redact` | 찾은 값을 화면에 가려서 표시 |

> `.githooks` 폴더에 두는 이유: 저장소에 함께 올라가서 팀원도 같은 hook을 받을 수 있습니다.

### ③ 실행 권한 주고 연결하기

```bash
chmod +x .githooks/pre-commit
git config core.hooksPath .githooks
```

- `chmod +x`: 파일을 프로그램처럼 실행할 수 있게 합니다.
- `git config core.hooksPath .githooks`: Git은 기본으로 `.git/hooks` 폴더만 봅니다. `.githooks` 폴더를 보게 바꿉니다.
- 이 설정은 **각자 컴퓨터의 저장소 설정**(`.git/config`)에 저장되고 GitHub에 올라가지 않습니다. 그래서 팀원도 clone한 뒤 직접 해야 합니다(10장).

### ④ 연결 확인

```bash
git config core.hooksPath
```

결과: `.githooks` → hook 폴더가 연결됨

```bash
ls -l .githooks/pre-commit
```

결과: `-rwxr-xr-x ... .githooks/pre-commit` → `x`가 있으면 실행 권한이 있음

### ⑤ 가짜 키로 시험하기

만든 장치는 실제로 막는지 확인하기 전에는 믿지 않습니다.

```bash
echo 'ECOS_API_KEY=FAKEKEY1234567890' > .env
echo 'url=https://ecos.bok.or.kr/api/StatisticSearch/FAKEKEY1234567890/json' > test-leak.txt
git add test-leak.txt
git commit -m "test"
```

결과: `[pre-commit] Blocked: the value of ECOS_API_KEY from .env is in the staged changes.` → 3겹 정상

시험 파일을 정리합니다.

```bash
git restore --staged test-leak.txt
rm test-leak.txt .env
```

### ⑥ 등록

```bash
git status
git add .githooks/pre-commit
git status
git commit -m "Add pre-commit hook to block API key leaks"
git push -u origin common/key-guard
gh pr create --base main --head common/key-guard --title "Add pre-commit hook to block API key leaks" --body "키 보호 3·4겹: commit 전 검사 hook"
gh pr merge common/key-guard --squash --delete-branch
git switch main
git pull
```

commit할 때 gitleaks 그림과 `INF no leaks found`가 나오면, 방금 만든 hook이 이 commit을 검사해서 통과시킨 것입니다.

### 막히면

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 시험에서 commit이 그냥 됨 | hook이 연결되지 않음 | ④ 연결 확인 |
| `Permission denied` | 실행 권한이 없음 | `chmod +x .githooks/pre-commit` |
| `gitleaks is not installed` | gitleaks가 없음 | 2-1 준비 확인 |
| 키가 아닌데 gitleaks에 걸림 | 오탐 | 화면을 공유하고 예외 등록 여부를 결정 |

---

## 8. 5겹: PR을 올리면 GitHub이 다시 검사한다

### 왜 필요한가

3·4겹은 **각자 컴퓨터**에서만 돕니다. 팀원이 hook을 설치하지 않았거나 `git commit --no-verify`로 건너뛰면 키가 그대로 올라갑니다. 그래서 **모두가 반드시 지나가는 곳**인 PR에서 한 번 더 검사합니다.

### 무엇을 검사하나

| 항목 | 내용 |
| --- | --- |
| 대상 | 저장소의 **commit 기록 전체** (첫 commit부터, PR에 새로 올라온 commit까지) |
| 방법 | 4겹과 같은 gitleaks 규칙 (키처럼 생긴 값) |
| 시점 | PR을 만들 때, PR에 새 commit을 올릴 때, main에 반영될 때 |
| 장소 | GitHub이 빌려주는 컴퓨터 (누구의 컴퓨터와도 상관없음) |
| 결과 | 통과하면 ✅, 발견하면 ❌와 함께 파일, 줄 번호, 걸린 규칙 표시 (값은 가림) |

- **기록 전체를 보는 이유**: 키가 든 파일을 나중에 지워도 과거 commit에는 남아 있기 때문입니다.
- **3겹 방식(실제 값 대조)을 쓰지 않는 이유**: GitHub에는 `.env`가 없습니다. 우리 키를 GitHub에 주지 않고, 줘서도 안 됩니다.

**4겹과의 차이**

| | 4겹 | 5겹 |
| --- | --- | --- |
| 장소 | 내 컴퓨터 | GitHub |
| 대상 | 이번 commit의 변경분 | commit 기록 전체 |
| 건너뛸 수 있나 | hook 미설치, `--no-verify`로 가능 | 누가 올리든 반드시 실행 |
| 성격 | 올라가기 전에 **막음** | 올라간 뒤에 **발견** |

### 따라하기

**① 사본 만들기**

```bash
git switch main
git pull
git switch -c common/pr-secret-scan
```

**② 검사 파일 만들기**

GitHub은 저장소의 `.github/workflows/` 폴더에 있는 파일을 읽고 그 내용대로 자동 검사를 돌립니다. 이 위치는 정해져 있습니다.

```bash
mkdir -p .github/workflows
```

```bash
cat > .github/workflows/secret-scan.yml <<'EOF'
name: secret-scan

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  gitleaks:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout (full history)
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Install gitleaks
        env:
          GH_TOKEN: ${{ github.token }}
          GITLEAKS_VERSION: v8.30.1
        run: |
          gh release download "$GITLEAKS_VERSION" -R gitleaks/gitleaks -p '*_linux_x64.tar.gz' -D /tmp/gitleaks
          tar -xzf /tmp/gitleaks/*.tar.gz -C /tmp/gitleaks
          sudo install /tmp/gitleaks/gitleaks /usr/local/bin/gitleaks

      - name: Scan all commits
        run: gitleaks git --redact --verbose
EOF
```

**검사 파일 해설**

| 부분 | 뜻과 이유 |
| --- | --- |
| `on: pull_request` | PR이 생기거나 바뀔 때 실행. 모두가 지나가는 곳이라서 |
| `push: branches: [main]` | main에 반영될 때도 실행. 원본이 깨끗한지 한 번 더 확인 |
| `permissions: contents: read` | 읽기만 허용. 검사에 필요한 만큼만 권한을 줌 |
| `runs-on: ubuntu-latest` | GitHub이 빌려주는 Ubuntu 컴퓨터에서 실행 |
| `fetch-depth: 0` | commit 기록 **전체**를 받음. 기본값은 마지막 commit만 받음 |
| `GH_TOKEN: ${{ github.token }}` | GitHub이 검사마다 자동으로 주는 임시 토큰. 우리가 만들 것은 없음 |
| `GITLEAKS_VERSION: v8.30.1` | dell-debian과 같은 버전. 내 컴퓨터와 GitHub의 결과를 같게 하려고 |
| `--redact` | 찾은 값을 로그에 그대로 보여주지 않음. 검사 로그로 키가 다시 새는 것을 막으려고 |

> YAML 파일은 들여쓰기(스페이스)가 뜻을 가집니다. 붙여 넣은 그대로 둡니다.

**③ 등록과 PR**

```bash
git status
git add .github/workflows/secret-scan.yml
git status
git commit -m "Add PR secret scan with gitleaks"
git push -u origin common/pr-secret-scan
gh pr create --base main --head common/pr-secret-scan --title "Add PR secret scan with gitleaks" --body "키 보호 5겹: PR 자동 검사"
```

**④ 검사 결과 확인**

```bash
gh pr checks common/pr-secret-scan --watch
```

결과: `All checks were successful` → 통과. 보통 1~2분 걸리고, 이 PR 자체가 5겹의 첫 시험입니다. PR 주소를 Windows 브라우저에서 열면 같은 결과를 볼 수 있습니다.

**⑤ 반영 (✅일 때만)**

```bash
gh pr merge common/pr-secret-scan --squash --delete-branch
git switch main
git pull
git log --oneline
```

맨 위가 `Add PR secret scan with gitleaks (#2)`이면 완료입니다.

### 막히면

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 검사가 아예 안 보임 | 파일 위치나 이름이 틀림 | `.github/workflows/`에 있는지, 확장자가 `.yml`인지 확인 |
| `Install gitleaks` 단계 실패 | 내려받기 문제 | 브라우저에서 PR의 Checks 탭 기록 확인 후 공유 |
| `Scan all commits` 단계 ❌ | 기록 어딘가에 키처럼 생긴 값이 있음 | **merge하지 않고** 11장으로 |

---

## 9. 실제 키 넣기

1·3·4·5겹을 모두 갖춘 뒤에 넣습니다. 장치보다 키가 먼저 들어가면, 그 사이의 실수는 막을 방법이 없습니다.

```bash
cp .env.example .env
nano .env
```

> 각 줄의 `=` 뒤에 사이트 마이페이지에서 복사한 키를 붙여 넣고 `Ctrl+O`(저장), `Enter`, `Ctrl+X`(종료)를 누릅니다.

**확인**

```bash
git status --ignored
```

결과: `Ignored files` 아래에 `.env`가 보이면 키 파일이 Git에서 제외된 상태입니다.

---

## 10. 팀원 시작 안내

새 팀원이 저장소를 받은 뒤 한 번만 합니다.

```bash
git clone https://github.com/schware/aidev-KEDP001.git /srv/projects/aidev-kedp001
cd /srv/projects/aidev-kedp001
git config core.hooksPath .githooks
gitleaks version
cp .env.example .env
```

| 명령 | 왜 |
| --- | --- |
| `git config core.hooksPath .githooks` | hook 파일은 저장소에 있지만 연결 설정은 각자 해야 함. 안 하면 3·4겹이 동작하지 않음 |
| `gitleaks version` | 없으면 hook이 commit을 막음 |
| `cp .env.example .env` | 견본을 복사해서 자기 키를 채움 |

---

## 11. 약속과 사고 대응

### 약속

- 키는 채팅, 메일, 문서, Copilot에 붙이지 않습니다.
- 키를 화면, 로그, 결과 보고에 표시하지 않습니다. (Agent 포함)
- hook에 걸리면 `--no-verify`로 우회하지 않고 보고합니다.
- 5겹 검사가 ❌면 merge하지 않습니다. 무료 비공개 저장소라 GitHub이 강제로 막지 못하므로 약속으로 지킵니다.

### 키가 새면

1. **키 재발급**: 해당 사이트에서 새 키를 받고 기존 키를 폐기합니다.
2. **`.env` 교체**: 새 키로 바꿉니다.
3. **기록 정리**: 그다음에 Git 기록을 정리합니다.

**키부터 바꾸는 이유**: 기록을 지우는 동안에도 이미 누군가 키를 가져갔을 수 있습니다. 키를 바꾸면 새어 나간 키는 그 순간 쓸모가 없어집니다.

---

## 12. 용어

| 용어 | 뜻 |
| --- | --- |
| `.env` | 실제 키를 적어 두는 파일. Git에 올리지 않음 |
| `.env.example` | 키 이름만 적은 견본. Git에 올림 |
| `.gitignore` | Git이 무시할 파일 목록 |
| hook | Git이 commit 같은 동작 직전에 자동으로 실행하는 검사 프로그램 |
| `core.hooksPath` | Git이 hook을 찾을 폴더를 지정하는 설정. 각자 컴퓨터에 저장됨 |
| gitleaks | 키처럼 생긴 값을 찾아 주는 검사 도구 |
| GitHub Actions | PR 등에 맞춰 GitHub이 자동으로 검사를 돌려 주는 기능 |
| branch | 원본을 건드리지 않고 작업하는 사본 |
| PR | "이 사본을 원본에 반영해 주세요"라는 결재 요청 |
| merge | 사본의 내용을 원본에 반영하는 것 |
