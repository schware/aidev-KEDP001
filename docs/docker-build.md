# KEDP Docker image 만들기 안내서

이 문서는 **image를 만들고(build) 실행 서버로 보내는 과정**을 자세히 설명한다.
- 대상 파일: `deploy/build.sh`, Dockerfile 3개, `.dockerignore`, `deploy/send.sh`
- 서버에서 띄우기, 백업, 되돌리기: `docs/docker-deploy.md`
- 배포 방식은 세 가지다(A, B, C). 이 문서의 1~11장은 A다. 비교와 PR 중심 배포 규칙은 12장(KEDP001-D025)

---

# 1. 큰 그림

```text
dell-debian (개발 서버)                                  macpro-debian (실행 서버)

 b-go-end/          ─┐                                    
 b-rust-processor/  ─┼─ deploy/build.sh ─→ image 3개 ─ deploy/send.sh ─→ image 3개
 b-proto/           ─┤     (docker build)   kedp-go         (save → gzip    kedp-go
 f-ts-end/          ─┤                      kedp-rust        → ssh → load)  kedp-rust
 deploy/nginx/      ─┘                      kedp-nginx                      kedp-nginx
                                                                              +
                                                                            postgres:17
                                                                    (Docker Hub에서 직접 받음)
```

- **image는 3개만 만든다.** go, rust, nginx(화면 포함)다.
- **PostgreSQL은 만들지 않는다.** 공식 image `postgres:17`을 macpro-debian이 Docker Hub에서 직접 받는다.
- **소스는 dell-debian에만 있다.** macpro-debian에는 image만 간다(KEDP001-D023).

---

# 2. 낱말

| 낱말 | 뜻 |
| --- | --- |
| image | 프로그램과 그 프로그램이 돌아가는 데 필요한 파일을 묶은 것. 실행하면 container가 된다 |
| Dockerfile | image를 만드는 순서를 적은 파일 |
| build | Dockerfile을 따라 image를 만드는 일(`docker build`) |
| build context | build할 때 Docker에게 넘겨주는 폴더. Dockerfile의 `COPY`는 이 폴더 안의 파일만 가져올 수 있다 |
| base image | Dockerfile의 `FROM`에 적는 출발점 image. 예: `golang:1.27-alpine` |
| stage | `FROM` 하나부터 다음 `FROM` 전까지. 한 Dockerfile에 여러 개 둘 수 있다 |
| multi-stage | 앞 stage에서 compile하고, 마지막 stage에는 결과 파일만 옮기는 방법. 실행 image가 작아진다 |
| scratch | 아무것도 없는 빈 image. shell도, `ls`도 없다 |
| layer | Dockerfile 한 줄(`RUN`, `COPY`)이 만드는 층. image는 layer를 쌓은 것이다 |
| cache | 이전 build의 layer를 다시 쓰는 것. 바뀐 것이 없으면 그 줄을 다시 실행하지 않는다 |
| tag | image 이름 뒤의 꼬리표. `kedp-go:17a8351`의 `17a8351` |
| `.dockerignore` | build context에서 빼는 파일 목록 |

---

# 3. `deploy/build.sh` 한 줄씩 읽기

## 3.1 쓰는 법

```bash
# 저장소 맨 위 폴더에서. 다른 Claude Code, docker 작업을 멈춘 뒤
deploy/build.sh                 # 3개 모두, tag는 git hash
deploy/build.sh 17a8351         # 3개 모두, tag를 직접 줌
deploy/build.sh 17a8351 nginx   # nginx만 다시 만듦
deploy/build.sh "" go           # go만, tag는 git hash
deploy/build.sh --allow-branch  # 시험용: branch나 -dirty에서도 만든다. 배포하지 않는다
```

**main에 merge된 commit에서만 만든다(KEDP001-D025 결론 9).** 지금 commit이 `origin/main`에 없거나 commit하지 않은 변경이 있으면 멈춘다. `--allow-branch`는 맨 앞에 붙인다.

## 3.2 내용

```sh
set -eu
```
- `-e`: 명령 하나라도 실패하면 거기서 멈춘다. go build가 실패했는데 rust로 넘어가는 일이 없다.
- `-u`: 정하지 않은 변수를 쓰면 멈춘다. 오타 때문에 빈 값으로 진행되는 것을 막는다.

```sh
root=$(cd "$(dirname "$0")/.." && pwd)
```
- 스크립트가 있는 `deploy/`의 한 단계 위, 곧 저장소 맨 위 폴더를 `root`로 잡는다.
- 그래서 어느 폴더에서 실행해도 경로가 맞는다.

```sh
allow_branch=
if [ "${1:-}" = --allow-branch ]; then
	allow_branch=1
	shift
fi

problem=
git -C "$root" merge-base --is-ancestor HEAD origin/main 2>/dev/null ||
	problem="commit ... is not in origin/main (git switch main && git pull)"
[ -z "$(git -C "$root" status --porcelain)" ] ||
	problem="${problem:+$problem; }uncommitted changes (git status)"
```
- **main 검사.** 배포할 수 있는 것은 PR에서 CI가 통과하고 main에 merge된 commit뿐이다(KEDP001-D025 결론 9).
- `merge-base --is-ancestor HEAD origin/main`: 지금 commit이 `origin/main`에 들어 있는지 본다. main의 예전 commit(되돌리기용)도 통과한다.
- `status --porcelain`: commit하지 않은 변경이 있는지 본다. 있으면 tag에 `-dirty`가 붙는 상태다.
- 하나라도 걸리면 `stop: ...`을 보이고 멈춘다. `--allow-branch`를 붙이면 `WARNING`만 보이고 만든다. 이 image는 시험용이고 배포하지 않는다.
- `origin/main`은 마지막 `git pull`(또는 `git fetch`) 때의 main이다. 방금 merge한 commit이 걸리면 `git pull`부터 한다.
- 2026-11-08에 pull하지 않은 채 PR branch에서 실행해 그 commit(`ff670bf`)으로 image가 만들어진 일이 있었다. 이 검사가 그것을 막는다.

```sh
tag=${1:-}
[ $# -gt 0 ] && shift
if [ -z "$tag" ]; then
	tag=$(git -C "$root" rev-parse --short HEAD)
	[ -z "$(git -C "$root" status --porcelain)" ] || tag="$tag-dirty"
fi
```
- 첫 번째 값이 tag다. 비어 있으면 **지금 commit의 짧은 hash**(예: `17a8351`)를 쓴다.
- 고친 파일이나 commit하지 않은 파일이 있으면 `-dirty`가 붙는다(예: `17a8351-dirty`).
- **왜 hash를 쓰나:** image만 보고도 어떤 소스로 만들었는지 알 수 있다. 문제가 생기면 `git show 17a8351`로 그때 코드를 본다.
- **`-dirty`가 붙으면:** commit에 없는 내용이 image에 들어갔다는 뜻이다. 배포용 image는 `git status`가 깨끗할 때 만든다.

```sh
case $tag in
	*[!A-Za-z0-9_.-]* | -* | .*) echo "invalid tag: $tag" >&2; exit 1 ;;
esac
```
- tag에는 영문, 숫자, `_`, `.`, `-`만 쓴다. `-`나 `.`로 시작하면 안 된다. Docker tag 규칙이다.

```sh
[ $# -gt 0 ] || set -- go rust nginx
```
- image 이름을 주지 않으면 `go rust nginx` 3개 모두를 만든다.

```sh
for name in "$@"; do
	...
	go)    docker build -t "kedp-go:$tag" "$root/b-go-end" ;;
	rust)  docker build -t "kedp-rust:$tag" --build-context proto="$root/b-proto" "$root/b-rust-processor" ;;
	nginx) docker build -t "kedp-nginx:$tag" --build-context nginx-conf="$root/deploy/nginx" "$root/f-ts-end" ;;
```
- **`for`로 하나씩 만든다.** 동시에 만들지 않는다. dell-debian 메모리가 작기 때문이다. 특히 Rust compile이 메모리를 많이 쓴다.
- `-t kedp-go:$tag`: 만든 image에 이름과 tag를 붙인다.
- 마지막 경로(`$root/b-go-end`)가 build context다. 그 폴더의 `Dockerfile`을 쓴다.
- `--build-context 이름=폴더`: build context 밖의 폴더를 하나 더 넘긴다. Dockerfile에서 `COPY --from=이름`으로 가져온다.
  - rust: `b-proto/`(Go와 Rust가 함께 쓰는 gRPC 약속서)가 필요하다.
  - nginx: `deploy/nginx/default.conf`(nginx 설정)가 필요하다.
  - 이 방법을 쓰면 build context를 저장소 전체로 넓히지 않아도 된다. 넘기는 파일이 적어 빠르다.

---

# 4. Dockerfile 3개

## 4.1 세 파일에 같은 것

```dockerfile
# syntax=docker/dockerfile:1
ARG GO_IMAGE=golang:1.27-alpine
FROM ${GO_IMAGE} AS build
...
FROM scratch
```
- `# syntax=docker/dockerfile:1`: 최신 Dockerfile 문법을 쓴다. `--build-context`를 쓰려면 필요하다.
- `ARG ..._IMAGE`: base image를 맨 위에 모아 두었다. 버전을 올릴 때 이 줄만 바꾼다.
- **두 stage로 나눈다.** `build` stage에서 compile하고, 실행 stage에는 결과만 옮긴다.
  - compile 도구(Go, Rust, Node)는 수백 MB다. 실행 image에는 들어가지 않는다.
  - 그래서 image가 작아지고 보내는 시간도 줄어든다.
- **`-alpine`:** Alpine Linux 기반의 작은 base image다.

## 4.2 Go: `b-go-end/Dockerfile`

```dockerfile
FROM ${GO_IMAGE} AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/kedp-server ./cmd/server
```

| 줄 | 하는 일 | 이유 |
| --- | --- | --- |
| `apk add ca-certificates` | HTTPS 인증서 묶음을 받는다 | Go가 ECOS, KOSIS, 공공데이터포털에 HTTPS로 접속한다. 인증서가 없으면 접속이 실패한다 |
| `COPY go.mod go.sum` → `go mod download` | 의존성 목록만 먼저 넣고 내려받는다 | **cache 때문이다.** 소스만 고치고 의존성이 그대로면 이 두 줄은 cache를 쓴다. 내려받기를 건너뛴다 |
| `COPY . .` | 나머지 소스를 넣는다 | `.dockerignore`에 적은 것은 빠진다(5장) |
| `CGO_ENABLED=0` | C library 없이 compile한다 | 다른 파일이 필요 없는 실행 파일 하나가 나온다. 그래서 빈 image(scratch)에서 돈다 |
| `-trimpath` | 실행 파일에서 build한 폴더 경로를 뺀다 | dell-debian의 경로가 image에 남지 않는다 |
| `-ldflags='-s -w'` | 디버그 정보를 뺀다 | 실행 파일이 작아진다 |

```dockerfile
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/kedp-server /kedp-server
ENV TZ=Asia/Seoul
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/kedp-server"]
```
- 실행 image에는 **인증서 파일과 실행 파일 2개만** 있다.
- `TZ=Asia/Seoul`: 수집 시각(06:00)과 방문자 날짜를 한국 시간으로 계산한다. 시간대 정보는 Go 실행 파일 안에 들어 있다(`time/tzdata`).
- `USER 65532:65532`: root가 아닌 번호로 실행한다. 프로그램이 뚫려도 할 수 있는 일이 적다. 그래서 관리자 계정 파일도 이 번호가 주인이다(`install -o 65532 -g 65532 -m 400`).
- `EXPOSE 8080`: 이 container가 8080을 쓴다는 **표시**일 뿐이다. 바깥에 여는 것은 compose의 `ports`가 한다. Go는 바깥에 열지 않는다.
- `ENTRYPOINT`: container가 시작되면 실행할 파일이다.

## 4.3 Rust: `b-rust-processor/Dockerfile`

```dockerfile
FROM ${RUST_IMAGE} AS build
RUN apk add --no-cache musl-dev protoc protobuf-dev
WORKDIR /src/b-rust-processor
COPY --from=proto . /src/b-proto
COPY Cargo.toml Cargo.lock build.rs ./
COPY src ./src
ENV CARGO_BUILD_JOBS=2
RUN cargo build --release --locked && strip target/release/kedp-processor
```

| 줄 | 하는 일 | 이유 |
| --- | --- | --- |
| `musl-dev` | Alpine용 C library | Rust 실행 파일을 다른 파일 없이 혼자 돌게(정적으로) 만든다. 그래서 scratch에서 돈다 |
| `protoc protobuf-dev` | proto compiler | `build.rs`가 `processor.proto`를 읽어 gRPC 코드를 만든다 |
| `COPY --from=proto . /src/b-proto` | `b-proto/`를 넣는다 | build.sh의 `--build-context proto=...`로 받은 폴더다. `build.rs`가 `../b-proto`를 찾으므로 폴더 위치를 저장소와 똑같이 맞췄다 |
| `CARGO_BUILD_JOBS=2` | 동시에 2개씩만 compile한다 | dell-debian 메모리 때문이다. 늘리면 빨라지지만 서버가 멈출 수 있다 |
| `--release` | 최적화한 실행 파일 | 실행이 빠르다. 대신 compile이 오래 걸린다 |
| `--locked` | `Cargo.lock`에 적힌 버전 그대로 쓴다 | 개발할 때와 같은 버전이 들어간다. lock이 맞지 않으면 실패한다 |
| `strip` | 디버그 정보를 뺀다 | 실행 파일이 작아진다 |

**Rust build가 가장 오래 걸리는 이유:** Go와 달리 의존성만 먼저 compile하는 줄이 없다. 그래서 `src`를 한 줄만 고쳐도 `cargo build` 줄의 cache가 깨진다. 그러면 tonic, tokio 같은 의존성까지 모두 다시 compile한다.

```dockerfile
FROM scratch
COPY --from=build /src/b-rust-processor/target/release/kedp-processor /kedp-processor
ENV TZ=Asia/Seoul
USER 65532:65532
EXPOSE 50051
ENTRYPOINT ["/kedp-processor"]
```
- 실행 image에는 **실행 파일 하나만** 있다.
- Rust는 바깥 인터넷에 접속하지 않으므로 인증서가 필요 없다. 접속하는 곳은 내부 network의 Go와 PostgreSQL뿐이다.
- 시각은 UTC로 계산하므로 시간대 파일이 필요 없다.
- 50051은 Go가 부르는 gRPC 포트다. 바깥에 열지 않는다.

## 4.4 nginx(화면): `f-ts-end/Dockerfile`

```dockerfile
FROM ${NODE_IMAGE} AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build
```
- 1단계는 Node로 화면을 만든다.
- `package.json`을 먼저 넣고 `npm ci`를 한다. Go의 `go mod download`와 같은 cache 방법이다.
- `npm ci`: `package-lock.json`에 적힌 버전 그대로 설치한다. lock이 `package.json`과 맞지 않으면 실패한다.
- `npm run build`: Vite가 React 소스를 `dist/` 폴더의 HTML, JS, CSS 파일로 만든다. 개발할 때 쓰는 Vite 서버는 image에 들어가지 않는다.

```dockerfile
FROM ${NGINX_IMAGE}
RUN apk add --no-cache tzdata
ENV TZ=Asia/Seoul
COPY --from=nginx-conf default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
```
- 2단계는 공식 nginx image에 두 가지만 더한다.
  - `deploy/nginx/default.conf`: `/api`는 Go로 넘기고 나머지는 화면 파일을 준다.
  - `dist/`: 1단계에서 만든 화면 파일
- nginx는 scratch가 아니다. nginx 프로그램이 자기 설정 파일과 library를 쓰기 때문이다.
- `tzdata`: nginx 기록(access log)의 시각을 한국 시간으로 남긴다.

---

# 5. `.dockerignore`: image에 넣지 않는 것

| 폴더 | 빼는 것 | 이유 |
| --- | --- | --- |
| `b-go-end` | `.env*`, `*.md`, `*_test.go`, `testdata` | 키가 든 `.env`가 image에 들어가면 안 된다. 테스트와 문서는 실행에 필요 없다 |
| `b-rust-processor` | `target`, `tests`, `scripts`, `.env*`, `*.md` | `target`은 dell-debian에서 compile한 결과다(수 GB). 넘기면 느리고, image 안에서 새로 compile한다 |
| `f-ts-end` | `node_modules`, `dist`, `.env*` | image 안에서 `npm ci`, `npm run build`로 새로 만든다 |

**`.env`를 빼는 것이 가장 중요하다.** image는 다른 서버로 보내는 파일이다. 키와 비밀번호는 image에 넣지 않는다(KEDP001-D002, D012). 실행할 때 macpro-debian의 `.env`에서 넣는다.

---

# 6. cache와 디스크

## 6.1 두 번째 build가 빠른 이유

Docker는 Dockerfile을 위에서부터 한 줄씩 본다. 그 줄과 그 줄이 쓰는 파일이 지난번과 같으면 지난번 결과(layer)를 그대로 쓴다. **한 줄이 바뀌면 그 아래 줄은 모두 다시 실행한다.**

```text
Go 소스 한 줄 고침
  COPY go.mod go.sum   → 같음, cache
  RUN go mod download  → 같음, cache   ← 내려받기 생략
  COPY . .             → 다름, 다시
  RUN go build         → 다시
```

그래서 자주 바뀌지 않는 것(의존성 목록)을 위에, 자주 바뀌는 것(소스)을 아래에 둔다.

cache를 쓰지 않고 처음부터 다시 만들려면 이렇게 한다.

```bash
docker build --no-cache -t kedp-go:17a8351 b-go-end
```

## 6.2 image가 쌓인다

build할 때마다 새 tag의 image가 남는다.
- **dell-debian:** build 중간 결과(cache)도 쌓인다.
- **macpro-debian:** 받은 image가 쌓인다.

```bash
docker image ls 'kedp-*'          # 크기, tag 확인
docker image rm kedp-go:<옛 tag> kedp-rust:<옛 tag> kedp-nginx:<옛 tag>
docker builder prune              # dell-debian: build cache 정리(다음 build는 느려진다)
docker system df                  # docker가 쓰는 디스크
```

macpro-debian에서는 **지금 쓰는 tag와 바로 전 tag(되돌리기용)는 남긴다.**

---

# 7. 만든 image 확인

```bash
docker image ls 'kedp-*'                  # 3개가 같은 tag로 있는지
docker history kedp-go:17a8351            # layer와 크기
docker image inspect kedp-go:17a8351 --format '{{.Config.User}} {{.Config.Entrypoint}}'
```

**go와 rust image는 scratch라 shell이 없다.** `docker exec -it <container> sh`로 들어갈 수 없다. 상태는 `docker compose logs go`로 본다.

---

# 8. `deploy/send.sh`: 보내기

```bash
deploy/send.sh macpro-debian 17a8351
```

```sh
images="kedp-nginx:$tag kedp-go:$tag kedp-rust:$tag"
for i in $images; do
	docker image inspect "$i" >/dev/null 2>&1 || { echo "no image: $i ..."; exit 1; }
done
docker save $images | gzip -1 | ssh "$host" 'gunzip | docker load'
```

| 단계 | 하는 일 |
| --- | --- |
| `docker image inspect` | 3개가 모두 있는지 먼저 본다. 하나라도 없으면 보내지 않는다 |
| `docker save` | image 3개를 tar 하나로 묶는다. 셋이 같이 쓰는 layer는 한 번만 들어간다 |
| `gzip -1` | 가장 빠른 단계로 압축한다. 압축률보다 속도를 택했다 |
| `ssh "$host"` | `~/.ssh/config`의 Host 이름(`macpro-debian`)으로 접속한다. 실제 주소는 저장소에 쓰지 않는다 |
| `gunzip \| docker load` | macpro-debian에서 압축을 풀고 image로 등록한다 |

- 파일을 디스크에 저장하지 않고 **흐름(pipe)으로 바로 보낸다.** 그래서 양쪽 모두 임시 파일이 남지 않는다.
- macpro-debian의 계정은 `docker` group에 있어야 한다(sudo 없이 `docker load`).
- registry(Docker Hub, GHCR)는 쓰지 않는다. 계정이나 token을 더 관리하지 않아도 된다(KEDP001-D023).

보낸 뒤에는 macpro-debian에서 tag를 바꾸고 다시 띄운다(`docs/docker-deploy.md` 7장).

```bash
# macpro-debian, /srv/projects/kedp
# .env의 KEDP_IMAGE_TAG=17a8351 로 바꾼 뒤
docker compose up -d
```

---

# 9. compose.yaml과 tag가 이어지는 곳

```yaml
  go:
    image: kedp-go:${KEDP_IMAGE_TAG:?KEDP_IMAGE_TAG is not set}
```
- compose는 `.env`의 `KEDP_IMAGE_TAG`로 어느 image를 띄울지 고른다.
- 값이 비어 있으면 `KEDP_IMAGE_TAG is not set`을 보이고 멈춘다.
- 그래서 **build.sh의 tag = send.sh의 tag = `.env`의 `KEDP_IMAGE_TAG`** 세 곳이 같아야 한다.
- 되돌리기는 `.env`의 tag만 전 값으로 바꾸고 `docker compose up -d` 하면 된다. 전 image가 남아 있어야 한다(6.2).

---

# 10. 한눈에 보는 순서

```bash
# dell-debian
# 1) 다른 Claude Code, docker 작업 멈춤
cd <저장소>
git switch main && git pull
git status                       # 깨끗해야 tag에 -dirty가 안 붙는다
deploy/build.sh                  # 마지막 줄 "== done: tag 1234abc" 의 tag를 적어 둔다
docker image ls 'kedp-*'
deploy/send.sh macpro-debian 1234abc

# macpro-debian
cd /srv/projects/kedp
# .env의 KEDP_IMAGE_TAG=1234abc
docker compose up -d
docker compose ps
```

DB 구조(migration)가 바뀌는 배포는 `docs/docker-deploy.md` 7장 순서를 따른다.

---

# 11. 문제가 생기면

| 보이는 것 | 이유 | 할 일 |
| --- | --- | --- |
| build 중 dell-debian이 느려지거나 멈춤 | 메모리 부족. Rust compile이 가장 많이 쓴다 | 다른 Claude Code, docker 작업을 멈추고 하나씩 만든다(`deploy/build.sh "" rust`) |
| `invalid tag` | tag에 쓸 수 없는 글자 | 영문, 숫자, `_ . -`만 쓴다 |
| `stop: commit ... is not in origin/main` | branch에 있거나 pull하지 않았다 | `git switch main && git pull` 뒤 다시 만든다. 시험용이면 `--allow-branch` |
| `stop: uncommitted changes` | commit하지 않은 변경이 있다 | `git status` 확인. 정리한 뒤 다시 만든다. 시험용이면 `--allow-branch` |
| tag 끝에 `-dirty` | `--allow-branch`로 변경이 있는 채 만들었다 | 시험용 image다. 배포하지 않는다 |
| `no image: kedp-...` (send.sh) | 그 tag로 만들지 않은 image가 있다 | `docker image ls 'kedp-*'`로 확인하고 빠진 것만 만든다 |
| `npm ci` 실패 | `package-lock.json`이 `package.json`과 맞지 않는다 | f-ts-end에서 `npm install`로 lock을 맞추고 commit한다 |
| `cargo build --locked` 실패 | `Cargo.lock`이 `Cargo.toml`과 맞지 않는다 | b-rust-processor에서 `cargo build`로 lock을 맞추고 commit한다 |
| base image를 받지 못함 | Docker Hub 접속 문제 | 잠시 뒤 다시 한다. 이미 받은 base image는 cache에서 쓴다 |
| `--build-context` 모름 | 오래된 Docker(BuildKit 없음) | `docker version`으로 확인. Docker 23 이상을 쓴다 |
| `no space left on device` | image, cache가 쌓였다 | 6.2의 정리 명령 |
| macpro-debian에서 `exec format error` | CPU 종류가 다른 곳에서 만든 image | 아래 "다른 컴퓨터에서 만들 때" |

## 다른 컴퓨터에서 만들 때

dell-debian과 macpro-debian은 둘 다 Intel/AMD 계열(amd64)이라 그대로 맞는다. Apple Silicon Mac(arm64)에서 image를 만들어 macpro-debian으로 보낼 때만 CPU 종류를 지정해야 한다.

```bash
docker build --platform linux/amd64 -t kedp-go:<tag> b-go-end
```

Mac에서 만든 image를 Mac에서 바로 돌릴 때는 필요 없다.

---

# 12. 세 가지 배포 방식

KEDP001-D025로 배포 방식을 세 가지로 둔다. 배포할 때마다 소유자가 하나를 고른다. **A가 기본이다.**

## 12.1 비교

| | A. dell-debian build | B. GitHub build + GHCR | C. GitHub build + artifact |
| --- | --- | --- | --- |
| image 만드는 곳 | dell-debian(`deploy/build.sh`) | GitHub Actions(`build-ghcr.yml`) | GitHub Actions(`build-artifact.yml`) |
| 옮기는 방법 | `docker save` → SSH → `docker load`(`deploy/send.sh`) | GHCR에 push, macpro-debian이 `docker pull` | artifact로 올리고, dell-debian이 받아 SSH → `docker load` |
| image 이름 | `kedp-go:<tag>` | `ghcr.io/schware/kedp-go:<tag>` | `kedp-go:<tag>` |
| `.env`의 `KEDP_IMAGE_PREFIX` | 비움 | `ghcr.io/schware/` | 비움 |
| 보관 | 서버에 남은 image | GHCR에 image마다 최근 5개 버전 | artifact 14일, 운영 중 + 새 버전 2개 |
| dell-debian 부담 | Rust compile(작업을 멈춘다) | 없음 | 받아서 보내기만 |
| 실행 서버의 비밀 | 없음 | 없음(package 공개) | 없음(dell-debian을 통로로) |

- **tag는 세 방식 모두 같다.** build한 main commit의 짧은 hash 7자리다(예: `c1bd8ab`). 같은 commit이면 어느 방식으로 만들어도 tag가 같다.
- **image 안의 내용도 같다.** 같은 Dockerfile과 build context(`proto`, `nginx-conf`)를 쓴다. 키와 비밀번호는 image에 넣지 않는다.
- A와 C는 image 이름이 같다(`kedp-go:<tag>`). 실행 서버에서는 똑같이 `docker load`한 image로 띄운다.

## 12.2 언제 무엇을 쓰나

| 상황 | 방식 |
| --- | --- |
| 평소. 개발 서버에서 바로 만들어 보낸다 | **A** |
| dell-debian에서 Rust compile을 돌리고 싶지 않다. 지난 버전 몇 개 안에서 되돌릴 수 있게 둔다 | **B** |
| QA까지 확정하고 배포한다. 되돌리기를 최소로 하고 14일 안에 확정한다 | **C** |

한 방식에 문제가 생기면 같은 tag를 다른 방식으로 만들어 쓸 수 있다.

## 12.3 PR 중심 배포 규칙(결론 9)

**PR로 main에 merge된 commit만 배포한다. branch별 배포는 하지 않는다.** 세 방식 모두 같다.

- 배포할 수 있는 것: PR에서 CI(test, secret-scan)가 통과하고 main에 merge된 commit
- main 보호 설정(소유자): main에 직접 push를 막고, PR과 CI 통과를 merge 조건으로 둔다
- A: `deploy/build.sh`가 `origin/main`에 없는 commit, `-dirty`에서 멈춘다(3장). 시험용 branch image는 `--allow-branch`로만 만들고 배포하지 않는다
- B, C: workflow가 실행할 commit이 main에 있고 merge된 PR의 commit인지 확인한다. 아니면 멈춘다. 실행 요약에 PR 번호를 적는다
- tag가 main commit의 hash라서, tag만 보고 main 어디에서 만든 버전인지 안다

**왜:** 배포한 버전마다 PR(무엇을, 왜 바꿨는지)과 CI 결과가 함께 남는다. branch image가 운영에 올라가면 main에 없는 코드가 돌고, 무엇이 운영 중인지 기록으로 추적할 수 없다.

## 12.4 B: `.github/workflows/build-ghcr.yml`

| 항목 | 내용 |
| --- | --- |
| 실행 | 손으로만(GitHub Actions 화면 → build-ghcr → Run workflow, branch는 main). main push마다 돌지 않는다 |
| 입력 `commit` | build할 main commit의 hash. 비우면 main 최신 commit |
| 검사 | main에서 실행했는지, commit이 main에 있는지, merge된 PR의 merge commit(squash 포함)인지. 아니면 멈춘다 |
| 권한 | `contents: read`, `packages: write`, `pull-requests: read`(commit에 연결된 PR 찾기). GHCR 로그인은 `GITHUB_TOKEN` |
| build | go → rust → nginx를 하나씩. Dockerfile, build context는 `deploy/build.sh`와 같다 |
| image | `ghcr.io/schware/kedp-go`, `kedp-rust`, `kedp-nginx` `:<tag>`. `latest`는 만들지 않는다 |
| label | `org.opencontainers.image.source`(저장소와 package를 잇는다), `org.opencontainers.image.revision`(commit) |
| 보관 | push한 뒤 image마다 tag 있는 최근 5개 버전만 남기고 나머지(같은 tag를 다시 올려 밀려난 버전 포함)를 지운다 |
| 실행 요약 | tag, commit, PR 번호, image 3개 이름, 지운 버전 |
| 쓰지 않는 것 | API 키, `.env` |

같은 commit으로 다시 실행하면 같은 tag를 덮어쓴다.

```bash
# dell-debian에서 실행하려면(gh 로그인됨)
gh workflow run build-ghcr.yml --ref main                       # main 최신 commit
gh workflow run build-ghcr.yml --ref main -f commit=<hash>      # 지정한 main commit
gh run list --workflow build-ghcr.yml --limit 3
```

**소유자가 한 번 할 일:** 첫 실행 뒤 GitHub 프로필 → Packages에서 `kedp-go`, `kedp-rust`, `kedp-nginx`를 각각 Package settings → Change visibility → **Public**으로 바꾼다. 그다음 Settings → Billing에서 Actions, Packages 비용이 0인지 확인한다(KEDP001-D025 결론 8).

배포 순서는 `docs/docker-deploy.md` 7장.

## 12.5 C: `.github/workflows/build-artifact.yml`, `deploy/send-artifact.sh`

### 원칙(KEDP001-D025 결론 5)

- **artifact는 14일 뒤 사라지는 것이 목적이다.** 14일 안에 QA와 확정을 끝낸다.
- **14일이 지난 버전으로는 되돌리지 않는다.** 문제가 생기면 고친 commit으로 새로 build한다(앞으로 가기).
- **두 버전이 살아 있게 운영한다.** 지금 운영 중인 버전과 새 버전이다. 새 버전을 build하기 전에 운영 중인 commit을 한 번 다시 build해 artifact를 새로 올린다. 그러면 되돌릴 버전의 14일이 다시 시작된다.
- **artifact 말고 다른 곳에 사본을 남기지 않는다.** dell-debian은 통로로만 쓰고, 받은 파일은 보낸 뒤 바로 지운다.
- 같은 commit을 다시 build하면 tag와 소스는 같다. base image가 그사이 갱신되었으면 image 안의 OS 파일은 조금 다를 수 있다.

**왜:** 많은 개발 조직에서 배포는 2주 안에 확정되고, 그보다 늦게 되돌린다면 그것 자체가 이상 신호다. 오래 보관하며 되돌리는 쓰임은 B가 맡는다.

### workflow

| 항목 | 내용 |
| --- | --- |
| 실행 | 손으로만(GitHub Actions 화면 → build-artifact → Run workflow, branch는 main) |
| 입력 `commit` | build할 main commit의 hash. 비우면 main 최신 commit |
| 검사 | B와 같다. main에서 실행했는지, commit이 main에 있는지, merge된 PR의 merge commit인지 |
| 권한 | `contents: read`, `pull-requests: read`(commit에 연결된 PR 찾기). artifact 올리기는 권한이 따로 필요 없다 |
| build | go → rust → nginx를 하나씩. 이름 `kedp-{go,rust,nginx}:<tag>`, Dockerfile, build context는 A와 같다 |
| 묶기 | `docker save` 3개 \| `gzip` → `kedp-images-<tag>.tar.gz` 하나. 셋이 같이 쓰는 layer는 한 번만 들어간다 |
| artifact | 이름 `kedp-images-<tag>`, 보관 14일 |
| 실행 요약 | tag, commit, PR 번호, artifact 이름, 파일 크기 |
| 쓰지 않는 것 | API 키, `.env` |

```bash
# dell-debian에서 실행하려면(gh 로그인됨)
gh workflow run build-artifact.yml --ref main                    # main 최신 commit
gh workflow run build-artifact.yml --ref main -f commit=<hash>   # 지정한 main commit
gh run list --workflow build-artifact.yml --limit 3
```

### `deploy/send-artifact.sh` (dell-debian)

```bash
deploy/send-artifact.sh macpro-debian <tag>
```

| 단계 | 하는 일 |
| --- | --- |
| tag 검사 | 짧은 hash 7자리(영문 소문자 a~f, 숫자)만 받는다 |
| artifact 찾기 | 이름이 `kedp-images-<tag>`이고 아직 사라지지 않은 것 중 가장 최근 것. 없으면 "14일이 지났거나 build하지 않은 tag다. 새로 build하라"를 보이고 멈춘다 |
| 받기 | `gh run download`로 임시 폴더(`mktemp -d`)에 받는다 |
| 보내기 | send.sh와 같은 방식. 압축한 채 SSH로 보내고 받는 서버에서 `gunzip \| docker load` |
| 확인 | 받는 서버에 `kedp-{nginx,go,rust}:<tag>` 3개가 있는지 본다 |
| 지우기 | 임시 폴더를 성공·실패·중단(Ctrl+C)과 상관없이 지운다(`trap ... EXIT`) |

- `gh`는 dell-debian에만 로그인되어 있다. **macpro-debian에는 token을 두지 않는다.** 공개 저장소라도 artifact를 받으려면 로그인이 필요해서 dell-debian을 통로로 쓴다.
- 받는 서버는 `~/.ssh/config`의 Host 이름으로 준다. 실제 주소는 쓰지 않는다.

배포 순서는 `docs/docker-deploy.md` 7장.
