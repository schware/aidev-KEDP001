// fakesource는 시험용 제공처 서버다. 제공처 요청을 받으면 samples/의 받아 둔 응답을 그대로 돌려준다.
// 실제 API를 부르지 않고 전체 흐름(수집 → Rust → DB)을 확인할 때 cmd/server -source-host와 함께 쓴다.
//
//	go run ./cmd/fakesource -addr 127.0.0.1:18080
//
// 고르는 샘플(요청 기간은 보지 않는다):
//   - ECOS StatisticSearch: samples/ecos/<통계 코드>-*-<주기>-*.json
//   - KOSIS statisticsParameterData: samples/kosis/<tblId>-*.json
//   - 공공데이터포털 getETFPriceInfo_V2: samples/data-go-kr/<종목 코드>-*.json
//
// 맞는 샘플이 없으면 404와 글 본문을 돌려준다(Rust 결과 ERROR HTTP_404).
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "들을 주소")
	samples := flag.String("samples", "../samples", "samples 폴더")
	flag.Parse()

	log.Printf("fakesource: listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler(*samples)))
}

func handler(samples string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dir, pattern := match(r)
		if dir == "" {
			http.Error(w, "unknown request", http.StatusNotFound)
			return
		}
		files, _ := filepath.Glob(filepath.Join(samples, dir, pattern))
		if len(files) == 0 {
			// 요청 주소에는 (가짜) 키가 들어 있어 기록하지 않는다.
			log.Printf("fakesource: %s %s: no sample", dir, pattern)
			http.Error(w, "no sample", http.StatusNotFound)
			return
		}
		body, err := os.ReadFile(files[0])
		if err != nil {
			http.Error(w, "read sample", http.StatusInternalServerError)
			return
		}
		log.Printf("fakesource: %s %s -> %s", dir, pattern, filepath.Base(files[0]))
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Write(body)
	})
}

// match는 요청에 맞는 샘플의 제공처 폴더와 파일 이름 pattern을 돌려준다. 모르는 요청이면 dir이 비어 있다.
func match(r *http.Request) (dir, pattern string) {
	q := r.URL.Query()
	switch p := r.URL.Path; {
	case strings.Contains(p, "/StatisticSearch/"):
		// /api/StatisticSearch/{키}/json/kr/{시작 행}/{끝 행}/{통계 코드}/{주기}/{시작}/{끝}/{항목 코드}…
		parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
		if len(parts) < 9 || !safe(parts[7]) || !safe(parts[8]) {
			return "", ""
		}
		return "ecos", parts[7] + "-*-" + parts[8] + "-*.json"
	case strings.HasSuffix(p, "/statisticsParameterData.do") && safe(q.Get("tblId")):
		return "kosis", q.Get("tblId") + "-*.json"
	case strings.HasSuffix(p, "/getETFPriceInfo_V2") && safe(q.Get("likeSrtnCd")):
		return "data-go-kr", q.Get("likeSrtnCd") + "-*.json"
	}
	return "", ""
}

// safe는 파일 이름 pattern에 넣어도 되는 값인지 알려 준다(영문, 숫자, _만).
func safe(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return false
		}
	}
	return true
}
