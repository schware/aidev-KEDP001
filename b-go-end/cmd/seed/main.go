// seed는 config/indicators.yaml의 지표 정의와 화면 처음 값을 DB(indicator, indicator_display)에 넣는다(KEDP001-D021, D022).
//
//	go run ./cmd/seed
//
// 이미 있는 지표(같은 id)는 고치지 않는다. 다시 실행해도 결과가 같고, 관리자 화면에서 바꾼 값을 덮어쓰지 않는다.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

func main() {
	envPath := flag.String("env", "../.env", ".env 파일 위치")
	indicatorsPath := flag.String("indicators", "config/indicators.yaml", "지표 정보 파일")
	flag.Parse()

	if err := run(*envPath, *indicatorsPath); err != nil {
		log.Fatal(err)
	}
}

func run(envPath, indicatorsPath string) error {
	entries, err := config.LoadIndicators(indicatorsPath)
	if err != nil {
		return err
	}
	if err := validate(entries); err != nil {
		return err
	}
	dbConfig, err := config.LoadDB(envPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	st, err := store.Open(ctx, dbConfig)
	if err != nil {
		return err
	}
	defer st.Close()

	r, err := st.Seed(ctx, entries)
	if err != nil {
		return err
	}
	log.Printf("indicator: added %d, kept %d", r.IndicatorAdded, r.IndicatorKept)
	log.Printf("indicator_display: added %d, kept %d", r.DisplayAdded, r.DisplayKept)
	return nil
}

// validate는 DB에 넣기 전에 파일만 보고 알 수 있는 잘못을 찾는다. 나머지(id 모양, 제공처별 칸, 주기)는 DB가 검사한다.
func validate(entries []config.Entry) error {
	if len(entries) == 0 {
		return errors.New("no indicators")
	}
	seen := map[string]bool{}
	for i, e := range entries {
		switch {
		case e.ID == "":
			return fmt.Errorf("indicators[%d]: id is empty", i)
		case seen[e.ID]:
			return fmt.Errorf("%s: id is duplicated", e.ID)
		case e.Display.ChartPeriod == "":
			return fmt.Errorf("%s: display.chart_period is empty", e.ID)
		}
		seen[e.ID] = true
	}
	return nil
}
