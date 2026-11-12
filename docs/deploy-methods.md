# KEDP 배포 방식 A, B, C

- 근거: KEDP001-D025
- 자세한 명령: `docs/docker-build.md` 12장, `docs/docker-deploy.md` 7장

## 1. 한눈에 보기

| | A | B | C |
| --- | --- | --- | --- |
| image 만드는 곳 | dell-debian | GitHub | GitHub |
| image 두는 곳 | dell-debian | GHCR | artifact |
| 누가 받을 수 있나 | - | **누구나**(로그인 없이) | **어떤 GitHub 계정이든** 로그인만 하면 |
| 보관 | 서버에 남은 image | 최근 5개 | 14일 |

세 방식 모두 같은 소스로 같은 image를 만든다.

## 2. 흐름

tag 값은 2026-11 실제로 쓴 값이다. A `c1bd8ab`(#77 merge), B `cab9c05`(#82 merge, PR #85), C `8ee99f4`(#83 merge, PR #86). 다음 배포 때는 그때의 main commit hash가 tag가 된다.

### A. dell-debian build

```text
 PR merge (main)
     │
     ▼
 dell-debian
   ① git switch main && git pull
   ② deploy/build.sh            main 검사 → image 3개 build (tag = commit hash → c1bd8ab)
   ③ deploy/send.sh macpro-debian c1bd8ab
     │
     │  SSH  (docker save → gzip → docker load)
     ▼
 macpro-debian
   ④ KEDP_IMAGE_TAG=c1bd8ab docker compose up -d
```

### B. GitHub build + GHCR

```text
 PR merge (main)
     │
     ▼
 GitHub Actions (build-ghcr)              ← Run workflow를 눌러야 돈다
   ① PR로 merge된 commit인지 확인
   ② image 3개 build (tag = commit hash → cab9c05)
   ③ GHCR에 push
   ④ 최근 5개만 남기고 정리
   ⑤ Summary에 tag, PR 번호
     │
     ▼
 GHCR  ghcr.io/schware/kedp-*:cab9c05       ← 공개, 로그인 없이 누구나 받음
     ▲
     │  docker pull
     │
 macpro-debian
   ⑥ curl … compose.yaml                  (compose.yaml이 바뀌었을 때만)
   ⑦ KEDP_IMAGE_PREFIX=ghcr.io/schware/ KEDP_IMAGE_TAG=cab9c05 docker compose pull go rust nginx
   ⑧ KEDP_IMAGE_PREFIX=ghcr.io/schware/ KEDP_IMAGE_TAG=cab9c05 docker compose up -d
```

### C. GitHub build + artifact

```text
 PR merge (main)
     │
     ▼
 GitHub Actions (build-artifact)          ← Run workflow를 눌러야 돈다
   ① PR로 merge된 commit인지 확인
   ② image 3개 build, 파일 하나로 묶기 (tag = commit hash → 8ee99f4)
   ③ artifact로 올리기  kedp-images-8ee99f4
   ④ Summary에 tag, PR 번호
     │
     ▼
 artifact  kedp-images-8ee99f4.tar.gz       ← 14일 뒤 사라짐, GitHub 로그인한 곳에서 받음
     ▲
     │  gh run download (로그인 필요)
     │
 dell-debian (GitHub에 로그인됨)
   ⑤ deploy/send-artifact.sh macpro-debian 8ee99f4
        받기 → SSH로 보내기 → 받은 파일 지우기
     │
     │  SSH  (docker load)
     ▼
 macpro-debian
   ⑥ KEDP_IMAGE_TAG=8ee99f4 docker compose up -d
```
## 3. 언제 무엇을 쓰나

| 상황 | 방식 |
| --- | --- |
| 평소처럼 개발 서버에서 바로 | **A** |
| dell-debian을 쉬게 하고, 최근 버전 몇 개 안에서 되돌리고 싶다 | **B** |
| QA까지 확정하고 배포한다. 14일 안에 확정하고, 지나면 되돌리지 않고 고쳐서 새로 만든다 | **C** |

## 4. 세 방식 모두 지키는 규칙

- PR로 main에 merge된 commit만 배포한다.
- tag는 그 commit의 짧은 hash 7자리다. 같은 commit이면 A, B, C의 tag가 같다.
- image 안에 키와 비밀번호는 없다. 키는 macpro-debian의 `.env`에서 넣는다.
- 비용은 0이다(공개 저장소, 공개 package).

## 5. 알아 둘 것

- C의 artifact가 사라져도 이미 macpro-debian에 받아 둔 image는 그대로 있다.
- macpro-debian에서 `gh auth login`으로 로그인하면 C도 macpro-debian에서 바로 받을 수 있다. D025는 실행 서버에 GitHub token을 두지 않으려고 dell-debian에서 받기로 정했다. 바꾸려면 D025 결론 5를 고친다.
- 명령 앞에 `KEDP_IMAGE_TAG=<tag>`를 붙이면 `.env`를 고치지 않고 그 실행에만 쓴다. 값 없이 `docker compose up -d`를 하면 `.env`에 적힌 버전으로 뜬다.

## 6. 처음 한 번 하는 GitHub 설정

배포 방식 B, C를 처음 갖출 때 한 번만 한다. KEDP001은 2026-11-12에 모두 끝냈다.

1. **Actions 등록**
   - `.github/workflows/`에 yml 파일을 넣고 PR로 main에 merge하면 GitHub 저장소 → Actions 왼쪽 목록에 자동으로 생긴다. 따로 등록 버튼은 없다.
   - KEDP001의 workflow: `test`, `secret-scan`(PR과 main merge 때 자동), `build-ghcr`, `build-artifact`(Run workflow를 눌러야 돈다)
   - 목록에 안 보이면 페이지를 새로 고친다.
2. **main 보호 설정(ruleset)**
   - Settings → Rules → Rulesets → New branch ruleset
   - Enforcement `Active`, Bypass list 비움, Target `Include default branch`(main)
   - Require a pull request before merging(approvals 0), Require status checks to pass(`go`, `rust`, `web`, `gitleaks`, source는 GitHub Actions)
   - 공개 저장소라 무료다.
3. **B의 package 공개**
   - build-ghcr를 처음 실행한 뒤 프로필 → Packages(또는 저장소 Code 화면 오른쪽 Packages)에서 `kedp-go`, `kedp-rust`, `kedp-nginx`
   - 각각 Package settings → Danger Zone → Change visibility → Public
4. **비용 확인**
   - 프로필 → Settings → Billing and licensing에서 Actions, Packages가 0원인지 본다.
5. **C를 받는 곳의 로그인**
   - artifact를 받을 서버에서 `gh auth login`을 한다. 지금은 dell-debian이다.