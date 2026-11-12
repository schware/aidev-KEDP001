#!/bin/sh
# 배포 C: GitHub artifact(kedp-images-<tag>)를 받아 실행 서버에 docker load한다(KEDP001-D025).
#
#   deploy/send-artifact.sh <SSH 설정 이름> <tag>
#
# dell-debian(gh 로그인됨)을 통로로만 쓴다. 받은 파일은 성공·실패와 상관없이 지운다.
# 실행 서버에는 token을 두지 않는다. 받는 서버는 ~/.ssh/config의 Host 이름으로 준다(KEDP001-D020).
# artifact는 14일 뒤 사라진다. 지난 버전으로는 되돌리지 않고 새로 build한다(결론 5).
set -eu

if [ $# -ne 2 ]; then
	echo "usage: $0 <ssh-host> <tag>" >&2
	exit 1
fi
host=$1
tag=$2
case $tag in
	*[!0-9a-f]* | "") echo "invalid tag: $tag (main commit의 짧은 hash 7자리)" >&2; exit 1 ;;
esac
[ ${#tag} -eq 7 ] || { echo "invalid tag: $tag (main commit의 짧은 hash 7자리)" >&2; exit 1; }

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
name="kedp-images-$tag"
images="kedp-nginx:$tag kedp-go:$tag kedp-rust:$tag"

# 그 이름의 artifact 중 아직 살아 있는(expired가 아닌) 가장 최근 것의 실행 번호
run_id=$(gh api "repos/{owner}/{repo}/actions/artifacts?name=$name&per_page=100" \
	--jq '[.artifacts[] | select(.expired | not)] | sort_by(.created_at) | last | .workflow_run.id // empty')
if [ -z "$run_id" ]; then
	echo "no artifact: $name" >&2
	echo "14일이 지났거나 build하지 않은 tag다. 새로 build하라: GitHub Actions → build-artifact → Run workflow" >&2
	exit 1
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'exit 1' HUP INT TERM

echo "== downloading $name (run $run_id)"
gh run download "$run_id" -n "$name" -D "$tmp"
file="$tmp/$name.tar.gz"
[ -f "$file" ] || { echo "no file in artifact: $name.tar.gz" >&2; exit 1; }
gzip -t "$file"

echo "== sending $images ($(du -h "$file" | cut -f1))"
# send.sh와 같은 방식: 압축한 채 보내고 받는 서버에서 풀어 load한다.
ssh "$host" 'gunzip | docker load' < "$file"
# shellcheck disable=SC2086
ssh "$host" docker image inspect $images >/dev/null || { echo "load check failed: $images" >&2; exit 1; }
echo "== loaded: $images"

echo "== done. 받는 서버에서 .env의 KEDP_IMAGE_PREFIX는 비우고 KEDP_IMAGE_TAG=$tag 로 바꾸고 docker compose up -d"
