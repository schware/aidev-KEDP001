// fakesource는 시험용 제공처 서버다. 제공처 요청을 받으면 samples/의 받아 둔 응답을 그대로 돌려준다.
// 실제 API를 부르지 않고 전체 흐름(수집 → Rust → DB)을 확인할 때 cmd/server -source-host와 함께 쓴다.
//
//	go run ./cmd/fakesource -addr 127.0.0.1:18080
//
// 고르는 샘플(요청 기간은 보지 않는다. 여럿이면 이름순 첫 파일이다):
//   - ECOS StatisticSearch: samples/ecos/<통계 코드>-<ITEM_CODE1>[-<ITEM_CODE2>…]-<주기>-*.json
//   - KOSIS statisticsParameterData: samples/kosis/<tblId>-<itmId>[-<objL1>-<objL2>…]-<prdSe>-*.json
//   - 공공데이터포털 getETFPriceInfo_V2: samples/data-go-kr/<종목 코드>-*.json
//
// 맞는 샘플이 없으면 404와 글 본문을 돌려준다(Rust 결과 ERROR HTTP_404). 통계 코드만 맞는 다른 파일로 대신 보내지 않는다.
package main

import (
	"flag"
	"fmt"
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
// 샘플 이름은 요청 값으로 만든다(source.Request.SampleName과 같은 규칙). 기간만 *로 둔다.
//
//	ECOS:  {통계 코드}-{ITEM_CODE1}[-{ITEM_CODE2}…]-{주기}-*.json
//	KOSIS: {tblId}-{itmId}[-{objL1}-{objL2}…]-{prdSe}-*.json
//	공공데이터포털: {종목 코드}-*.json
//
// pattern에 넣는 값은 영문, 숫자, _만 받는다. 그 밖의 글자가 있으면 모르는 요청이다(404).
func match(r *http.Request) (dir, pattern string) {
	q := r.URL.Query()
	switch p := r.URL.Path; {
	case strings.Contains(p, "/StatisticSearch/"):
		// /api/StatisticSearch/{키}/json/kr/{시작 행}/{끝 행}/{통계 코드}/{주기}/{시작}/{끝}/{ITEM_CODE1}/{ITEM_CODE2}…
		parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
		if len(parts) < 12 {
			return "", ""
		}
		stat, cycle, items := parts[7], parts[8], parts[11:]
		if n := name(stat, items, cycle); n != "" {
			return "ecos", n
		}
	case strings.HasSuffix(p, "/statisticsParameterData.do"):
		var classes []string
		for i := 1; q.Has(fmt.Sprintf("objL%d", i)); i++ {
			classes = append(classes, q.Get(fmt.Sprintf("objL%d", i)))
		}
		if n := name(q.Get("tblId"), append([]string{q.Get("itmId")}, classes...), q.Get("prdSe")); n != "" {
			return "kosis", n
		}
	case strings.HasSuffix(p, "/getETFPriceInfo_V2") && safe(q.Get("likeSrtnCd")):
		return "data-go-kr", q.Get("likeSrtnCd") + "-*.json"
	}
	return "", ""
}

// name은 {표}-{코드…}-{주기}-*.json pattern이다. 값 하나라도 비었거나 safe가 아니면 빈 글자다.
func name(table string, codes []string, cycle string) string {
	parts := append(append([]string{table}, codes...), cycle)
	for _, v := range parts {
		if !safe(v) {
			return ""
		}
	}
	return strings.Join(parts, "-") + "-*.json"
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
