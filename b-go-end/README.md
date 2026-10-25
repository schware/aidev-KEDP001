# b-go-end

Go 수집 모듈. 자세한 구조와 결정은 `docs/dev-complete/b-go-end.md`를 본다.

| 명령 | 하는 일 |
| --- | --- |
| `cmd/server` | `config/indicators.yaml`에 등록된 지표 하나의 정상 응답을 받아 저장한다 |
| `cmd/sample` | 요청 값을 바로 넣어 부르고, 받은 응답을 그대로 저장한다(조사용 샘플) |

명령은 모두 `b-go-end` 폴더에서 실행한다. 키는 `../.env`에서 읽는다(`-env`로 바꿀 수 있다).

## 샘플 수집 도구 (`cmd/sample`)

- `indicators.yaml`에 없는 표·항목·분류·주기·기간도 그대로 넣어 부른다.
- 정상, 오류, 데이터 없음 응답을 모두 그대로 저장한다. 응답을 받지 못했을 때(키 없음, 시간 초과)만 실패다.
- 응답마다 `<이름>.request.json`에 요청 정보(키를 `***`로 가린 주소, HTTP 상태, 받은 시각)를 함께 저장한다.
- 저장 폴더 `-out`은 꼭 넣는다(기본값 없음). 그 아래에 제공처 폴더(`ecos/`, `kosis/`, `data-go-kr/`)를 만든다.
- 이미 있는 파일은 덮어쓰지 않는다. 같은 요청을 다시 받으려면 `-out` 폴더나 `-name`을 바꾼다.
- 조사 샘플은 `../samples/repository/` 아래에 둔다(git에 올리지 않는다).
- 목록 파일의 요청은 부르기 전에 모두 확인한다. 하나라도 틀리면 아무것도 부르지 않는다.

### 요청 종류

| 제공처 `-source` | 요청 종류 `-api` | 꼭 넣는 값 | 기본 파일 이름 |
| --- | --- | --- | --- |
| `ecos` | `search` (기본, StatisticSearch) | `-stat -item -cycle -start -end` | `통계코드-항목-분류…-주기-시작-끝.json` |
| `ecos` | `items` (StatisticItemList) | `-stat` | `통계코드-items.json` |
| `kosis` | `data` (기본, statisticsParameterData) | `-stat -item -cycle -start -end` | `tblId-itmId-objL…-주기-시작-끝.json` |
| `kosis` | `meta` (getMeta) | `-stat -meta-type` | `tblId-meta-type.json` |
| `data-go-kr` | `price` (기본, getETFPriceInfo_V2) | `-stat -start -end` (YYYYMMDD) | `종목코드-D-시작-끝.json` |

- `-stat`: ECOS 통계 코드, KOSIS `tblId`, 공공데이터포털 종목 코드
- `-item`: ECOS `ITEM_CODE1`, KOSIS `itmId`
- `-classes`: 쉼표로 나눈 분류 코드. KOSIS `objL1,objL2…`, ECOS `ITEM_CODE2,ITEM_CODE3…`
- `-cycle`: 제공처 요청 값 그대로(ECOS `A/Q/M/D`, KOSIS `Y/Q/M`). 공공데이터포털은 넣지 않는다.
- `-meta-type`: KOSIS getMeta `type` 값 그대로(예: `ITM`, `TBL`, `PRD`)
- `-name`: 저장 파일 이름(폴더 없이). `.json`이 없으면 붙인다.
- KOSIS `orgId`는 `101`(통계청)로 고정이다.

### 예시: 요청 하나

```sh
# ECOS 통계표의 항목 목록
go run ./cmd/sample -out ../samples/repository/check -source ecos -api items -stat 722Y001

# KOSIS 분류가 2개인 표, 분기
go run ./cmd/sample -out ../samples/repository/check \
  -source kosis -stat DT_XXXX -item T10 -classes 00,1 -cycle Q -start 2024 -end 2025

# KOSIS 메타(항목·분류)
go run ./cmd/sample -out ../samples/repository/check -source kosis -api meta -stat DT_1DA7001S -meta-type ITM
```

저장 결과:

```text
../samples/repository/check/ecos/722Y001-items.json           응답 본문 그대로
../samples/repository/check/ecos/722Y001-items.request.json   요청 정보
```

```json
{
  "source": "ecos",
  "api": "items",
  "url": "https://ecos.bok.or.kr/api/StatisticItemList/***/json/kr/1/1000/722Y001",
  "status": 200,
  "received_at": "2026-10-25T16:00:00+09:00"
}
```

### 예시: 목록 파일로 여러 요청

목록 파일에는 키를 적지 않는다. 모르는 칸이 있으면 멈춘다.

```yaml
# requests.yaml
requests:
  - source: ecos
    stat: 722Y001
    item: "0101000"
    cycle: M
    start: "190001"
    end: "190003"
    name: nodata-722Y001   # 데이터 없음 응답 확인
  - source: kosis
    api: meta
    stat: DT_1DA7001S
    meta_type: ITM
  - source: data-go-kr
    stat: "069500"
    start: "20261020"
    end: "20261022"
```

```sh
go run ./cmd/sample -out ../samples/repository/check -list requests.yaml
```

## 테스트

실제 API를 부르지 않는다. `../samples/`의 받아 둔 응답만 읽는다.

```sh
go test ./...
```
