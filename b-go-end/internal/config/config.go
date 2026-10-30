// Package config는 .env의 API 키·DB 접속 정보·Rust 주소와 지표 정보(indicators.yaml)를 읽는다.
package config

import (
	"bufio"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Keys는 제공처별 API 키다. 값을 출력하거나 기록하지 않는다.
type Keys struct {
	ECOS     string
	KOSIS    string
	DataGoKr string
}

// All은 비어 있지 않은 키를 모두 돌려준다. 키 가리기에 쓴다.
func (k Keys) All() []string {
	var out []string
	for _, v := range []string{k.ECOS, k.KOSIS, k.DataGoKr} {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// LoadKeys는 path의 .env 파일에서 키를 읽는다.
// 같은 이름의 환경 변수가 있으면 환경 변수를 쓴다. 파일이 없어도 오류가 아니다.
func LoadKeys(path string) (Keys, error) {
	get, err := lookup(path)
	if err != nil {
		return Keys{}, err
	}
	return Keys{
		ECOS:     get("ECOS_API_KEY"),
		KOSIS:    get("KOSIS_API_KEY"),
		DataGoKr: get("DATA_GO_KR_API_KEY"),
	}, nil
}

// DBUser는 Go가 DB에 접속하는 계정이다(KEDP001-D022 14). 비밀번호는 같은 이름의 변수에 둔다.
const DBUser = "kedp_go"

// DB는 DB 접속 정보다. 값을 출력하거나 기록하지 않는다(%v로 찍어도 나오지 않는다).
type DB struct {
	Host     string
	Port     uint16
	Name     string
	Password string
}

func (DB) String() string { return "DB{***}" }

// LoadDB는 path의 .env 파일과 환경 변수에서 DB 접속 정보를 읽는다. 하나라도 비어 있으면 오류다.
// 오류에는 변수 이름만 넣는다.
func LoadDB(path string) (DB, error) {
	get, err := lookup(path)
	if err != nil {
		return DB{}, err
	}
	var db DB
	var missing []string
	for _, v := range []struct {
		name string
		dst  *string
	}{{"KEDP_DB_HOST", &db.Host}, {"KEDP_DB_NAME", &db.Name}, {DBUser, &db.Password}} {
		if *v.dst = get(v.name); *v.dst == "" {
			missing = append(missing, v.name)
		}
	}
	port := get("KEDP_DB_PORT")
	if port == "" {
		missing = append(missing, "KEDP_DB_PORT")
	}
	if len(missing) > 0 {
		return DB{}, fmt.Errorf("%s is not set", strings.Join(missing, ", "))
	}
	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil || p == 0 {
		return DB{}, fmt.Errorf("KEDP_DB_PORT is not a port number")
	}
	db.Port = uint16(p)
	return db, nil
}

// DefaultProcessorAddr는 KEDP_PROCESSOR_ADDR가 비어 있을 때 쓰는 Rust 처리 서버 주소다(b-rust-processor와 같다).
const DefaultProcessorAddr = "127.0.0.1:50051"

// LoadProcessorAddr는 Rust 처리 서버 주소(KEDP_PROCESSOR_ADDR)를 읽는다. 비어 있으면 DefaultProcessorAddr다.
func LoadProcessorAddr(path string) (string, error) {
	get, err := lookup(path)
	if err != nil {
		return "", err
	}
	if addr := get("KEDP_PROCESSOR_ADDR"); addr != "" {
		return addr, nil
	}
	return DefaultProcessorAddr, nil
}

// DefaultAPIAddr는 KEDP_API_ADDR가 비어 있을 때 쓰는 조회 API 주소다.
const DefaultAPIAddr = "127.0.0.1:8080"

// LoadAPIAddr는 조회 API 주소(KEDP_API_ADDR)를 읽는다. 비어 있으면 DefaultAPIAddr다.
func LoadAPIAddr(path string) (string, error) {
	get, err := lookup(path)
	if err != nil {
		return "", err
	}
	if addr := get("KEDP_API_ADDR"); addr != "" {
		return addr, nil
	}
	return DefaultAPIAddr, nil
}

// Admin은 관리자 API 설정이다(KEDP001-D018). 값을 출력하거나 기록하지 않는다.
type Admin struct {
	// AllowIPs는 관리자 API를 쓸 수 있는 주소다. 비어 있으면 관리자 API는 모두 403이다.
	AllowIPs []netip.Prefix
	// File은 계정 파일 위치다. 비어 있으면 로그인할 수 없다.
	File string
}

func (Admin) String() string { return "Admin{***}" }

// LoadAdmin은 KEDP_ADMIN_ALLOW_IPS(쉼표로 나눈 IP 또는 CIDR)와 KEDP_ADMIN_FILE을 읽는다.
// 틀린 주소가 있으면 오류다. 오류에는 변수 이름만 넣는다.
func LoadAdmin(path string) (Admin, error) {
	get, err := lookup(path)
	if err != nil {
		return Admin{}, err
	}
	a := Admin{File: get("KEDP_ADMIN_FILE")}
	for _, v := range strings.Split(get("KEDP_ADMIN_ALLOW_IPS"), ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		p, err := parsePrefix(v)
		if err != nil {
			return Admin{}, fmt.Errorf("KEDP_ADMIN_ALLOW_IPS has an invalid IP or CIDR")
		}
		a.AllowIPs = append(a.AllowIPs, p)
	}
	return a, nil
}

// parsePrefix는 IP(한 주소) 또는 CIDR을 읽는다. IPv4에 대응한 IPv6 주소는 IPv4로 바꾼다.
func parsePrefix(v string) (netip.Prefix, error) {
	if strings.Contains(v, "/") {
		p, err := netip.ParsePrefix(v)
		if err != nil {
			return netip.Prefix{}, err
		}
		if p.Addr().Is4In6() && p.Bits() >= 96 {
			p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
		}
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(v)
	if err != nil {
		return netip.Prefix{}, err
	}
	a = a.Unmap()
	return netip.PrefixFrom(a, a.BitLen()), nil
}

// lookup은 이름으로 값을 찾는 함수를 돌려준다. 환경 변수가 비어 있지 않으면 환경 변수, 아니면 .env 파일 값이다.
// 파일이 없어도 오류가 아니다.
func lookup(path string) (func(name string) string, error) {
	env, err := readEnvFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return func(name string) string {
		if v, ok := os.LookupEnv(name); ok && v != "" {
			return v
		}
		return env[name]
	}, nil
}

func readEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		env[strings.TrimSpace(name)] = value
	}
	// 오류 메시지에 줄 내용(키 값)을 넣지 않는다.
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	return env, nil
}

// Indicator는 지표 하나의 수집 정보다. DB indicator 테이블의 칼럼과 같다.
type Indicator struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
	// OrgID는 KOSIS 기관 코드(orgId)다. 다른 제공처는 비운다.
	OrgID    string `yaml:"org_id"`
	StatCode string `yaml:"stat_code"`
	ItemCode string `yaml:"item_code"`
	// Classes는 분류 코드다. KOSIS는 objL1, objL2… 순서로 넣는다. ECOS는 ITEM_CODE2, ITEM_CODE3… 순서로 넣는다(지금 등록된 ECOS 지표는 비워 둔다).
	Classes []string `yaml:"classes"`
	// ValueField는 공공데이터포털에서 지표 값으로 쓰는 항목이다(clpr, KEDP001-D022 8). 다른 제공처는 비운다.
	ValueField string `yaml:"value_field"`
	// Cycle은 D/M/Q/A다. 연은 A이고, KOSIS에 요청할 때만 Y로 바꾼다(KEDP001-D022 3).
	Cycle string `yaml:"cycle"`
}

// Display는 지표 하나의 화면 처음 값이다. DB indicator_display 테이블의 칼럼과 같다.
type Display struct {
	Visible bool `yaml:"visible"`
	Order   int  `yaml:"order"`
	// ChartPeriod는 상세 화면 차트 기간이다. PostgreSQL interval 글(예: 10 years)이다.
	// 관측값이 없을 때 처음 수집할 과거 범위도 이 기간이다(KEDP001-D022 12).
	ChartPeriod string `yaml:"chart_period"`
	// Notice는 상세 화면 맨 위 안내 문구다. 없으면 비운다.
	Notice string `yaml:"notice"`
}

// Entry는 indicators.yaml의 지표 하나다. DB에 넣을 처음 값이다(cmd/seed).
type Entry struct {
	Indicator `yaml:",inline"`
	Display   Display `yaml:"display"`
}

type indicatorFile struct {
	Indicators []Entry `yaml:"indicators"`
}

// LoadIndicators는 indicators.yaml을 읽는다.
func LoadIndicators(path string) ([]Entry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f indicatorFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return f.Indicators, nil
}
