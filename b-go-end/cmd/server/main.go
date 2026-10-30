// server는 KEDP 수집 서버다. DB indicator의 지표를 수집 시각(app_setting collect_time, Asia/Seoul)마다 수집하고,
// 받은 응답을 Rust 처리 서버(ProcessPage)로 넘긴다(KEDP001-D019, D022, b-proto/README.md).
// 사용자 화면의 조회 API(KEDP_API_ADDR, internal/api)와 관리자 API(/api/admin/, internal/admin)도 함께 연다.
//
//	go run ./cmd/server          # 조회 API를 열고, 수집 시각마다 수집한다(trigger schedule)
//	go run ./cmd/server -once    # 지금 한 번 수집하고 끝낸다(trigger manual). 조회 API는 열지 않는다
//
// 시작할 때 Rust Check가 실패하면 멈춘다(KEDP001-D020).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/admin"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/api"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/collect"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	datagokr "github.com/schware/aidev-KEDP001/b-go-end/internal/source/data_go_kr"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/kosis"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// tick은 수집 시각이 지났는지 확인하는 간격이다.
const tick = 30 * time.Second

func main() {
	envPath := flag.String("env", "../.env", ".env 파일 위치")
	once := flag.Bool("once", false, "지금 한 번 수집하고 끝낸다(trigger manual)")
	timeout := flag.Duration("timeout", 20*time.Second, "제공처 요청 시간 제한")
	sourceHost := flag.String("source-host", "", "시험용: 제공처 대신 요청을 보낼 주소(예: http://127.0.0.1:18080, cmd/fakesource)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *envPath, *once, *timeout, *sourceHost); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, envPath string, once bool, timeout time.Duration, sourceHost string) error {
	keys, err := config.LoadKeys(envPath)
	if err != nil {
		return err
	}
	dbConfig, err := config.LoadDB(envPath)
	if err != nil {
		return err
	}
	addr, err := config.LoadProcessorAddr(envPath)
	if err != nil {
		return err
	}
	apiAddr, err := config.LoadAPIAddr(envPath)
	if err != nil {
		return err
	}
	adminConfig, err := config.LoadAdmin(envPath)
	if err != nil {
		return err
	}
	accounts := admin.Accounts{}
	if adminConfig.File != "" {
		if accounts, err = admin.LoadAccounts(adminConfig.File); err != nil {
			return err
		}
	}

	client := fetch.New(timeout, keys.All()...)
	if sourceHost != "" {
		if err := client.SetHost(sourceHost); err != nil {
			return err
		}
		log.Printf("test mode: requests go to %s instead of the providers", sourceHost)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.New("KEDP_PROCESSOR_ADDR is invalid")
	}
	defer conn.Close()
	processor := kedpv1.NewProcessorServiceClient(conn)
	if err := check(ctx, processor); err != nil {
		return err
	}

	st, err := store.Open(ctx, dbConfig)
	if err != nil {
		return err
	}
	defer st.Close()

	c := &collect.Collector{
		Store:     st,
		Processor: processor,
		Sources: map[string]source.Source{
			"ecos":       ecos.New(keys.ECOS, client),
			"kosis":      kosis.New(keys.KOSIS, client),
			"data-go-kr": datagokr.New(keys.DataGoKr, client),
		},
		HasKey: client.HasKey,
	}

	if once {
		sum, err := c.Run(ctx, store.TriggerManual)
		log.Printf("collection finished (%s)", sum)
		return err
	}

	// 주소를 먼저 열어 두어, 열지 못하면(이미 쓰는 포트 등) 바로 멈춘다. 오류에 주소를 넣지 않는다.
	ln, err := net.Listen("tcp", apiAddr)
	if err != nil {
		return errors.New("cannot open API address (KEDP_API_ADDR)")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	apiErr := make(chan error, 1)
	if len(adminConfig.AllowIPs) == 0 {
		log.Printf("admin API: KEDP_ADMIN_ALLOW_IPS is empty; every admin request gets 403")
	}
	if len(accounts) == 0 {
		log.Printf("admin API: no admin account (KEDP_ADMIN_FILE); login always fails")
	}
	mux := http.NewServeMux()
	mux.Handle("/api/admin/", admin.Handler(admin.Config{
		Store:          st,
		AllowIPs:       adminConfig.AllowIPs,
		TrustedProxies: adminConfig.TrustedProxies,
		Accounts:       accounts,
		KeySet:         map[string]bool{"ecos": keys.ECOS != "", "kosis": keys.KOSIS != "", "data-go-kr": keys.DataGoKr != ""},
		Loc:            collect.Seoul,
	}))
	mux.Handle("/", api.Handler(st, time.Now, collect.Seoul))
	go func() {
		apiErr <- api.Serve(ctx, ln, mux)
		cancel() // API가 멈추면 수집도 멈춘다
	}()

	log.Printf("server started (API and schedule)")
	err = c.Schedule(ctx, tick)
	cancel()
	if e := <-apiErr; e != nil {
		return fmt.Errorf("API server: %w", e)
	}
	return err
}

// check는 Rust 처리 서버의 Check를 부른다. Rust가 DB에 접속할 수 있을 때만 성공이다(b-proto/README.md).
func check(ctx context.Context, p kedpv1.ProcessorServiceClient) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := p.Check(ctx, &kedpv1.CheckRequest{}); err != nil {
		return fmt.Errorf("processor check failed (KEDP_PROCESSOR_ADDR): %v", collect.GRPCError(err))
	}
	log.Printf("processor check ok")
	return nil
}
