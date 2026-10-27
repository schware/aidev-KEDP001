# ECOS 샘플

요청 주소의 키 자리는 `***`로 가렸다. 받은 날: 2026-10-25 (일·분기·연 샘플은 2026-10-26)

모든 요청의 앞부분은 같다: `StatisticSearch/***/json/kr/`

| 파일 | 내용 | 요청 (앞부분 뒤) |
| --- | --- | --- |
| `722Y001-0101000-D-20261001-20261023.json` | 한국은행 기준금리, 일, 정상 응답 23행 (`TIME` `20261001`~`20261023`) | `1/1000/722Y001/D/20261001/20261023/0101000` |
| `722Y001-0101000-M-202401-202609.json` | 한국은행 기준금리, 월, 정상 응답 33행 | `1/1000/722Y001/M/202401/202609/0101000` |
| `722Y001-0101000-Q-2025Q1-2025Q3.json` | 한국은행 기준금리, 분기, 정상 응답 3행 (`TIME` `2025Q1`~`2025Q3`) | `1/1000/722Y001/Q/2025Q1/2025Q3/0101000` |
| `722Y001-0101000-A-2023-2025.json` | 한국은행 기준금리, 연, 정상 응답 3행 (`TIME` `2023`~`2025`) | `1/1000/722Y001/A/2023/2025/0101000` |
| `error-INFO-100.json` | 인증키가 유효하지 않음 | 틀린 키로 요청 |
| `error-INFO-200.json` | 해당하는 데이터가 없음 | `1/10/722Y001/M/190001/190003/0101000` |

- 응답에는 주기가 없다. 기간 형식은 주기마다 다르다(설계서 5.1).
- 일 자료는 주말에도 행이 온다(2026-10-01~23 요청에 23행).

월 응답은 `b-go-end`에서 받았다.

```sh
cd b-go-end
go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
```

일·분기·연 응답은 기간 형식을 확인하려고 샘플 수집 도구로 받았다.

```sh
cd b-go-end
go run ./cmd/sample -out ../samples/repository/16-period -source ecos -stat 722Y001 -item 0101000 -cycle D -start 20261001 -end 20261023
go run ./cmd/sample -out ../samples/repository/16-period -source ecos -stat 722Y001 -item 0101000 -cycle Q -start 2025Q1 -end 2025Q3
go run ./cmd/sample -out ../samples/repository/16-period -source ecos -stat 722Y001 -item 0101000 -cycle A -start 2023 -end 2025
```
