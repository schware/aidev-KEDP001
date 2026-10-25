// server는 KEDP 수집 프로그램의 시작점이다.
// 1단계(KEDP001-D021)에서는 지표 하나의 응답 샘플을 받아 저장하는 일만 한다.
//
//	go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
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
	default:
		return fmt.Errorf("source %q is not implemented", ind.Source)
	}

	body, err := src.Fetch(context.Background(), source.Request{Indicator: ind, Start: start, End: end})
	if err != nil {
		return err
	}
	// 정상 응답만 샘플로 저장한다.
	rows, err := ecos.Parse(body)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-%s-%s-%s-%s.json", ind.StatCode, ind.ItemCode, ind.Cycle, start, end)
	path, err := client.Save(outDir+"/"+src.Name(), name, body)
	if err != nil {
		return err
	}
	log.Printf("saved %d rows to %s", len(rows), path)
	return nil
}
