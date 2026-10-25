# ECOS 샘플

요청 주소의 키 자리는 `***`로 가렸다. 받은 날: 2026-10-25

| 파일 | 내용 | 요청 |
| --- | --- | --- |
| `722Y001-0101000-M-202401-202609.json` | 한국은행 기준금리, 월, 정상 응답 33행 | `StatisticSearch/***/json/kr/1/1000/722Y001/M/202401/202609/0101000` |
| `error-INFO-100.json` | 인증키가 유효하지 않음 | 틀린 키로 요청 |
| `error-INFO-200.json` | 해당하는 데이터가 없음 | `StatisticSearch/***/json/kr/1/10/722Y001/M/190001/190003/0101000` |

정상 응답은 `b-go-end`에서 받았다.

```sh
cd b-go-end
go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
```
