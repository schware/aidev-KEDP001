// server는 KEDP 수집 프로그램의 시작점이다.
// 지금(KEDP001-D021 3단계)은 지표 하나의 응답 샘플을 받아 저장하는 일만 한다.
//
//	go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	datagokr "github.com/schware/aidev-KEDP001/b-go-end/internal/source/data_go_kr"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/kosis"
)

func main() {
	envPath := flag.String("env", "../.env", ".env 파일 위치")
	indicatorsPath := flag.String("indicators", "config/indicators.yaml", "지표 정보 파일")
	id := flag.String("indicator", "", "지표 id (indicators.yaml)")
	start := flag.String("start", "", "조회 시작 기간")
	end := flag.String("end", "", "조회 끝 기간")
	outDir := flag.String("out", "../samples", "샘플 저장 폴더 (아래에 제공처 폴더를 만든다)")
	timeout := flag.Duration("timeout", 20*time.Second, "요청 시간 제한")
	flag.Parse()

	if *id == "" || *start == "" || *end == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*envPath, *indicatorsPath, *id, *start, *end, *outDir, *timeout); err != nil {
		log.Fatal(err)
	}
}

func run(envPath, indicatorsPath, id, start, end, outDir string, timeout time.Duration) error {
	keys, err := config.LoadKeys(envPath)
	if err != nil {
		return err
	}
	list, err := config.LoadIndicators(indicatorsPath)
	if err != nil {
		return err
	}
	ind, err := config.FindIndicator(list, id)
	if err != nil {
		return err
	}

	client := fetch.New(timeout, keys.All()...)
	var src source.Source
	switch ind.Source {
	case "ecos":
		src = ecos.New(keys.ECOS, client)
	case "kosis":
		src = kosis.New(keys.KOSIS, client)
	case "data-go-kr":
		src = datagokr.New(keys.DataGoKr, client)
	default:
		return fmt.Errorf("source %q is not implemented", ind.Source)
	}

	req := source.Request{Indicator: ind, Start: start, End: end}
	body, err := src.Fetch(context.Background(), req)
	if err != nil {
		return err
	}
	// 정상 응답만 샘플로 저장한다.
	n, err := src.Check(body)
	if err != nil {
		return err
	}

	path, err := client.Save(filepath.Join(outDir, src.Name()), req.SampleName(), body)
	if err != nil {
		return err
	}
	log.Printf("saved %d rows to %s", n, path)
	return nil
}
