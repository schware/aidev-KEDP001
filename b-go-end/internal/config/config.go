// Package config는 .env의 API 키와 지표 정보(indicators.yaml)를 읽는다.
package config

import (
	"bufio"
	"fmt"
	"os"
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
	env, err := readEnvFile(path)
	if err != nil && !os.IsNotExist(err) {
		return Keys{}, err
	}
	get := func(name string) string {
		if v, ok := os.LookupEnv(name); ok && v != "" {
			return v
		}
		return env[name]
	}
	return Keys{
		ECOS:     get("ECOS_API_KEY"),
		KOSIS:    get("KOSIS_API_KEY"),
		DataGoKr: get("DATA_GO_KR_API_KEY"),
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

// Indicator는 지표 하나의 수집 정보다.
type Indicator struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Source   string `yaml:"source"`
	StatCode string `yaml:"stat_code"`
	ItemCode string `yaml:"item_code"`
	Cycle    string `yaml:"cycle"`
}

type indicatorFile struct {
	Indicators []Indicator `yaml:"indicators"`
}

// LoadIndicators는 indicators.yaml을 읽는다.
func LoadIndicators(path string) ([]Indicator, error) {
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

// FindIndicator는 id가 같은 지표를 찾는다.
func FindIndicator(list []Indicator, id string) (Indicator, error) {
	for _, ind := range list {
		if ind.ID == id {
			return ind, nil
		}
	}
	return Indicator{}, fmt.Errorf("indicator %q not found", id)
}
