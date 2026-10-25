// Package fetch는 외부 API 요청, 시간 제한, 키 가리기, 응답 저장을 맡는다.
// 키를 주소에 붙인 요청은 이 패키지를 거쳐 보내고, 기록할 때는 Mask를 거친 문자열만 쓴다(KEDP001-D012의 2겹).
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const masked = "***"

// forms는 secret이 글에 나타날 수 있는 형태를 모두 돌려준다.
// 원래 형태, 주소에 들어갈 때 바뀌는 형태(인코딩), 그리고 secret이 이미 인코딩된 형태일 때(공공데이터포털 Encoding 키) 푼 형태다.
func forms(secret string) []string {
	out := []string{secret, url.PathEscape(secret), url.QueryEscape(secret)}
	if strings.Contains(secret, "%") {
		if raw, err := url.QueryUnescape(secret); err == nil && raw != secret {
			out = append(out, raw, url.PathEscape(raw), url.QueryEscape(raw))
		}
	}
	return out
}

// Mask는 s 안의 키를 모두 ***로 바꾼다. 주소에 들어갈 때 바뀌는 형태(인코딩)도 함께 가린다.
func Mask(s string, secrets ...string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, form := range forms(secret) {
			s = strings.ReplaceAll(s, form, masked)
		}
	}
	return s
}

// StatusError는 HTTP 상태 코드가 200이 아닌 응답이다.
// 제공처에 따라 오류 내용이 본문에 오므로(공공데이터포털) 본문을 함께 둔다. URL은 가린 주소다.
type StatusError struct {
	URL  string
	Code int
	Body []byte
}

func (e *StatusError) Error() string { return fmt.Sprintf("GET %s: status %d", e.URL, e.Code) }

// Client는 키를 알고 있는 HTTP 클라이언트다. 오류와 기록에서 키를 가린다.
type Client struct {
	http    *http.Client
	secrets []string
}

// New는 시간 제한이 timeout인 Client를 만든다. secrets는 가릴 키 목록이다.
func New(timeout time.Duration, secrets ...string) *Client {
	return &Client{http: &http.Client{Timeout: timeout}, secrets: secrets}
}

// Mask는 이 Client가 아는 키를 가린다.
func (c *Client) Mask(s string) string { return Mask(s, c.secrets...) }

// Get은 rawURL에 GET 요청을 보내고 응답 본문을 돌려준다.
// 돌려주는 오류에는 가린 주소만 들어 있다. 원래 오류는 키가 들어 있어 감싸지 않는다.
// 상태 코드가 200이 아니면 *StatusError를 돌려준다.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	log.Printf("GET %s", c.Mask(rawURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.New(c.Mask(err.Error()))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errors.New(c.Mask(err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(c.Mask(fmt.Sprintf("read %s: %v", rawURL, err)))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{URL: c.Mask(rawURL), Code: resp.StatusCode, Body: body}
	}
	return body, nil
}

// Save는 응답 본문을 dir/name에 저장한다. 본문에 키가 (어떤 형태로든) 들어 있으면 저장하지 않는다.
func (c *Client) Save(dir, name string, body []byte) (string, error) {
	if c.Mask(string(body)) != string(body) {
		return "", errors.New("response contains an API key; not saved")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
