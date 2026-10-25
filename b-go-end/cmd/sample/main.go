// sample은 요청 값을 바로 넣어 제공처 API를 부르고, 받은 응답을 그대로 파일로 저장한다(Issue #22).
// 정상, 오류, 데이터 없음 응답을 모두 저장하고, 응답마다 요청 정보(키를 가린 주소, HTTP 상태, 받은 시각)를 옆에 저장한다.
//
//	go run ./cmd/sample -out ../samples/repository/check -source ecos -stat 722Y001 -item 0101000 -cycle M -start 202401 -end 202403
//	go run ./cmd/sample -out ../samples/repository/check -list requests.yaml
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	datagokr "github.com/schware/aidev-KEDP001/b-go-end/internal/source/data_go_kr"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/kosis"
)

func main() {
	envPath := flag.String("env", "../.env", ".env 파일 위치")
	outDir := flag.String("out", "", "저장 폴더 (꼭 넣는다. 아래에 제공처 폴더를 만든다)")
	listPath := flag.String("list", "", "요청 목록 파일(YAML). 넣으면 아래 요청 값은 쓰지 않는다")
	timeout := flag.Duration("timeout", 20*time.Second, "요청 시간 제한")
	var r Request
	var classes string
	flag.StringVar(&r.Source, "source", "", "제공처: ecos | kosis | data-go-kr")
	flag.StringVar(&r.API, "api", "", "요청 종류. ecos: search(기본) | items, kosis: data(기본) | meta, data-go-kr: price(기본)")
	flag.StringVar(&r.Stat, "stat", "", "ECOS 통계 코드, KOSIS tblId, 공공데이터포털 종목 코드")
	flag.StringVar(&r.Item, "item", "", "ECOS ITEM_CODE1, KOSIS itmId")
	flag.StringVar(&classes, "classes", "", "분류 코드, 쉼표로 나눈다. KOSIS objL1,objL2…, ECOS ITEM_CODE2,ITEM_CODE3…")
	flag.StringVar(&r.Cycle, "cycle", "", "주기 (제공처 요청 값 그대로. ECOS A/Q/M/D, KOSIS Y/Q/M)")
	flag.StringVar(&r.Start, "start", "", "조회 시작 기간")
	flag.StringVar(&r.End, "end", "", "조회 끝 기간")
	flag.StringVar(&r.MetaType, "meta-type", "", "KOSIS getMeta type (예: ITM, TBL, PRD)")
	flag.StringVar(&r.Name, "name", "", "저장 파일 이름 (기본: 요청 값으로 만든다)")
	flag.Parse()

	if *outDir == "" || (*listPath == "") == (r.Source == "") {
		fmt.Fprintln(os.Stderr, "-out과, -list 또는 -source 중 하나를 넣는다.")
		flag.Usage()
		os.Exit(2)
	}
	if classes != "" {
		r.Classes = strings.Split(classes, ",")
	}
	reqs := []Request{r}
	if *listPath != "" {
		var err error
		if reqs, err = loadList(*listPath); err != nil {
			log.Fatal(err)
		}
	}
	if err := prepare(reqs); err != nil {
		log.Fatal(err)
	}
	if err := run(*envPath, *outDir, reqs, *timeout); err != nil {
		log.Fatal(err)
	}
}

type sources struct {
	ecos     *ecos.Source
	kosis    *kosis.Source
	dataGoKr *datagokr.Source
}

// fetch는 요청 종류에 맞는 제공처 요청을 보낸다.
func (s sources) fetch(ctx context.Context, r Request) ([]byte, error) {
	switch r.API {
	case apiSearch:
		return s.ecos.Fetch(ctx, r.sourceRequest())
	case apiItems:
		return s.ecos.FetchItems(ctx, r.Stat)
	case apiData:
		return s.kosis.Fetch(ctx, r.sourceRequest())
	case apiMeta:
		return s.kosis.FetchMeta(ctx, r.Stat, r.MetaType)
	case apiPrice:
		return s.dataGoKr.Fetch(ctx, r.sourceRequest())
	}
	return nil, fmt.Errorf("api %q is not implemented", r.API)
}

func run(envPath, outDir string, reqs []Request, timeout time.Duration) error {
	keys, err := config.LoadKeys(envPath)
	if err != nil {
		return err
	}
	client := fetch.New(timeout, keys.All()...)
	s := sources{
		ecos:     ecos.New(keys.ECOS, client),
		kosis:    kosis.New(keys.KOSIS, client),
		dataGoKr: datagokr.New(keys.DataGoKr, client),
	}

	failed := 0
	for i, r := range reqs {
		if err := fetchAndSave(context.Background(), client, s, outDir, r); err != nil {
			log.Printf("request %d: %v", i+1, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d requests failed", failed, len(reqs))
	}
	return nil
}

// fetchAndSave는 요청을 보내고 받은 응답을 상태 코드와 상관없이 저장한다.
// 응답을 받지 못했을 때(키 없음, 시간 초과 등)만 오류다.
func fetchAndSave(ctx context.Context, client *fetch.Client, s sources, outDir string, r Request) error {
	var got *fetch.Exchange
	ctx = fetch.WithRecorder(ctx, func(e fetch.Exchange) { got = &e })
	_, fetchErr := s.fetch(ctx, r)
	if got == nil {
		return fetchErr
	}
	path, err := save(client, filepath.Join(outDir, r.Source), r, *got)
	if err != nil {
		return err
	}
	log.Printf("saved %s (status %d)", path, got.Status)
	if fetchErr != nil {
		// 오류 응답도 저장했다. 제공처가 알려 준 오류만 보여 준다.
		log.Printf("  %v", fetchErr)
	}
	return nil
}

// requestInfo는 응답 옆에 저장하는 요청 정보다. URL은 키를 가린 주소다.
type requestInfo struct {
	Source     string `json:"source"`
	API        string `json:"api"`
	URL        string `json:"url"`
	Status     int    `json:"status"`
	ReceivedAt string `json:"received_at"`
}

// save는 응답 본문을 dir/<이름>에, 요청 정보를 dir/<이름>.request.json에 저장한다.
// 이미 있는 파일은 덮어쓰지 않는다. 본문에 키가 들어 있으면 둘 다 저장하지 않는다(fetch.Client.Save).
func save(client *fetch.Client, dir string, r Request, e fetch.Exchange) (string, error) {
	for _, name := range []string{r.Name, r.infoName()} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return "", fmt.Errorf("%s already exists; not overwritten", filepath.Join(dir, name))
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	info, err := json.MarshalIndent(requestInfo{
		Source:     r.Source,
		API:        r.API,
		URL:        client.Mask(e.URL),
		Status:     e.Status,
		ReceivedAt: e.ReceivedAt.Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return "", err
	}
	path, err := client.Save(dir, r.Name, e.Body)
	if err != nil {
		return "", err
	}
	if _, err := client.Save(dir, r.infoName(), append(info, '\n')); err != nil {
		return "", err
	}
	return path, nil
}
