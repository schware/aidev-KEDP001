#!/bin/sh
# image 3개(kedp-go, kedp-rust, kedp-nginx)를 하나씩 만든다. 동시에 만들지 않는다(KEDP001-D023, dell-debian 메모리).
# 다른 무거운 작업(Rust 빌드, 다른 docker build)이 돌고 있지 않을 때 실행한다.
#
#   deploy/build.sh [tag] [go|rust|nginx ...]
#
# tag를 비우면 git commit의 짧은 hash다(고친 파일이 있으면 -dirty가 붙는다).
# 이름을 주면 그 image만 만든다.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
tag=${1:-}
[ $# -gt 0 ] && shift
if [ -z "$tag" ]; then
	tag=$(git -C "$root" rev-parse --short HEAD)
	[ -z "$(git -C "$root" status --porcelain)" ] || tag="$tag-dirty"
fi
case $tag in
	*[!A-Za-z0-9_.-]* | -* | .*) echo "invalid tag: $tag" >&2; exit 1 ;;
esac
[ $# -gt 0 ] || set -- go rust nginx

for name in "$@"; do
	echo "== kedp-$name:$tag ($(date '+%H:%M:%S'))"
	case $name in
		go) docker build -t "kedp-go:$tag" "$root/b-go-end" ;;
		rust) docker build -t "kedp-rust:$tag" --build-context proto="$root/b-proto" "$root/b-rust-processor" ;;
		nginx) docker build -t "kedp-nginx:$tag" --build-context nginx-conf="$root/deploy/nginx" "$root/f-ts-end" ;;
		*) echo "unknown image: $name (go, rust, nginx)" >&2; exit 1 ;;
	esac
done
echo "== done: tag $tag"
