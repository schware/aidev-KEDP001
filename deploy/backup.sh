#!/bin/sh
# kedp DB 백업(KEDP001-D020). 실행 서버의 compose 폴더(compose.yaml, .env가 있는 곳)에 두고 crontab에서 부른다.
# postgres container에서 pg_dump로 kedp를 받아 KEDP_BACKUP_DIR/kedp-YYYYMMDD.dump(하루 1개)로 둔다.
# 14일보다 오래된 파일은 지운다. 같은 날 다시 돌리면 그날 파일을 새로 만든다.
set -eu
umask 077
cd "$(dirname "$0")"

# .env에서 KEDP_BACKUP_DIR만 읽는다(.env 전체를 읽어 들이지 않는다).
dir=$(sed -n 's/^KEDP_BACKUP_DIR=//p' .env | tail -n 1 | sed -e 's/^["'\'']//' -e 's/["'\'']$//')
case $dir in
	/*) ;;
	*) echo "KEDP_BACKUP_DIR is not an absolute path" >&2; exit 1 ;;
esac
mkdir -p "$dir"

today=$(TZ=Asia/Seoul date +%Y%m%d)
out="$dir/kedp-$today.dump"
tmp="$out.tmp"
trap 'rm -f "$tmp"' EXIT

# 계정과 비밀번호는 container 안의 POSTGRES_USER를 쓴다(container 안의 연결이라 비밀번호를 묻지 않는다).
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d kedp -Fc' >"$tmp"
[ -s "$tmp" ] || { echo "backup is empty" >&2; exit 1; }
mv "$tmp" "$out"

cutoff=$(TZ=Asia/Seoul date -d '14 days ago' +%Y%m%d)
for f in "$dir"/kedp-*.dump; do
	[ -e "$f" ] || continue
	d=${f##*/kedp-}
	d=${d%.dump}
	case $d in
		'' | *[!0-9]*) continue ;;
	esac
	if [ "$d" -lt "$cutoff" ]; then
		rm -f -- "$f"
	fi
done
echo "backup: $out ($(wc -c <"$out") bytes)"
