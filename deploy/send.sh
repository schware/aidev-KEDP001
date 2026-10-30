#!/bin/sh
# image 3개를 실행 서버로 보낸다: docker save → SSH → docker load(KEDP001-D023).
#
#   deploy/send.sh <SSH 설정 이름> <tag>
#
# 받는 서버는 ~/.ssh/config의 Host 이름으로 준다. 실제 주소는 저장소에 쓰지 않는다(KEDP001-D020).
# 받는 서버의 계정은 docker를 쓸 수 있어야 한다(docker group).
set -eu

if [ $# -ne 2 ]; then
	echo "usage: $0 <ssh-host> <tag>" >&2
	exit 1
fi
host=$1
tag=$2
images="kedp-nginx:$tag kedp-go:$tag kedp-rust:$tag"

for i in $images; do
	docker image inspect "$i" >/dev/null 2>&1 || { echo "no image: $i (deploy/build.sh $tag)" >&2; exit 1; }
done

echo "== sending $images"
# shellcheck disable=SC2086
docker save $images | gzip -1 | ssh "$host" 'gunzip | docker load'
echo "== done. 받는 서버에서 .env의 KEDP_IMAGE_TAG=$tag 로 바꾸고 docker compose up -d"
