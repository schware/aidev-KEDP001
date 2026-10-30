package admin

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// 관리자 인증(KEDP001-D018): 허용 IP + 계정 파일의 계정·비밀번호 + 서버 메모리 session.

const (
	// CookieName은 session cookie 이름이다.
	CookieName = "kedp_admin_session"
	// CookiePath는 session cookie를 보내는 경로다.
	CookiePath = "/api/admin"
	// SessionTTL은 로그인 뒤 session이 유지되는 시간이다. 지나면 다시 로그인한다.
	SessionTTL = 8 * time.Hour

	// MinPasswordLen은 비밀번호의 최소 글자 수다.
	MinPasswordLen = 12
	// MaxPasswordBytes는 bcrypt가 쓰는 비밀번호의 최대 바이트 수다. 넘으면 뒷부분이 버려지므로 받지 않는다.
	MaxPasswordBytes = 72
	// HashCost는 cmd/adminpass가 쓰는 bcrypt cost다.
	HashCost = 12
)

var accountPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// ValidAccount는 계정 이름이 영문, 숫자, '.', '_', '-'로 된 1~64자인지 알려 준다.
func ValidAccount(name string) bool { return accountPattern.MatchString(name) }

// CheckPassword는 새 비밀번호의 길이를 검사한다(cmd/adminpass).
func CheckPassword(pw string) error {
	if utf8.RuneCountInString(pw) < MinPasswordLen {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	if len(pw) > MaxPasswordBytes {
		return fmt.Errorf("password must be at most %d bytes", MaxPasswordBytes)
	}
	return nil
}

// HashLine은 계정 파일에 넣을 한 줄("계정:bcrypt hash")을 만든다.
func HashLine(account, pw string) (string, error) {
	if !ValidAccount(account) {
		return "", errors.New("account must be 1-64 characters of A-Z, a-z, 0-9, '.', '_', '-'")
	}
	if err := CheckPassword(pw); err != nil {
		return "", err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), HashCost)
	if err != nil {
		return "", err
	}
	return account + ":" + string(h), nil
}

// Accounts는 계정 이름별 bcrypt hash다.
type Accounts map[string][]byte

// LoadAccounts는 계정 파일을 읽는다. 한 줄에 "계정:bcrypt hash" 하나다. 빈 줄과 #으로 시작하는 줄은 건너뛴다.
// 오류에는 파일 위치와 줄 내용(hash)을 넣지 않고 줄 번호만 넣는다.
func LoadAccounts(path string) (Accounts, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot read the admin account file (KEDP_ADMIN_FILE)")
	}
	defer f.Close()
	out := Accounts{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, hash, ok := strings.Cut(line, ":")
		if !ok || !ValidAccount(name) {
			return nil, fmt.Errorf("admin account file (KEDP_ADMIN_FILE) line %d: want account:bcrypt-hash", n)
		}
		if _, err := bcrypt.Cost([]byte(hash)); err != nil {
			return nil, fmt.Errorf("admin account file (KEDP_ADMIN_FILE) line %d: not a bcrypt hash", n)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("admin account file (KEDP_ADMIN_FILE) line %d: duplicate account", n)
		}
		out[name] = []byte(hash)
	}
	if err := sc.Err(); err != nil {
		return nil, errors.New("cannot read the admin account file (KEDP_ADMIN_FILE)")
	}
	return out, nil
}

// dummyHash는 없는 계정으로 로그인할 때도 비밀번호 비교 시간을 같게 하려고 쓰는 hash다.
var dummyHash = sync.OnceValue(func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("kedp-dummy-password"), HashCost)
	if err != nil {
		panic(err)
	}
	return h
})

// verify는 계정과 비밀번호가 맞는지 알려 준다. 계정이 없어도 비밀번호 비교를 한 번 한다.
func (a Accounts) verify(account, pw string) bool {
	hash, ok := a[account]
	if !ok {
		hash = dummyHash()
	}
	err := bcrypt.CompareHashAndPassword(hash, []byte(pw))
	return ok && err == nil
}

// session은 로그인한 관리자 하나다.
type session struct {
	account string
	expires time.Time
}

// sessions는 서버 메모리의 session 목록이다. 서버를 다시 켜면 비워진다.
type sessions struct {
	mu sync.Mutex
	m  map[string]session
}

// create는 새 session을 만들고 token(무작위 32바이트)을 돌려준다. 끝난 session은 이때 지운다.
func (s *sessions) create(account string, now time.Time) (string, session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	ss := session{account: account, expires: now.Add(SessionTTL)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[string]session{}
	}
	for k, v := range s.m {
		if !now.Before(v.expires) {
			delete(s.m, k)
		}
	}
	s.m[token] = ss
	return token, ss, nil
}

// get은 token의 session을 찾는다. 없거나 끝났으면 false다.
func (s *sessions) get(token string, now time.Time) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.m[token]
	if !ok {
		return session{}, false
	}
	if !now.Before(ss.expires) {
		delete(s.m, token)
		return session{}, false
	}
	return ss, true
}

func (s *sessions) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, token)
}

// ClientIP는 요청한 쪽의 주소다. RemoteAddr를 쓰고, RemoteAddr가 믿을 proxy(trusted) 안에 있을 때만
// X-Forwarded-For의 마지막 주소를 쓴다. 주소를 읽을 수 없으면 false다.
func ClientIP(remoteAddr string, forwardedFor []string, trusted []netip.Prefix) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return netip.Addr{}, false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	addr = addr.Unmap()
	if !allowed(trusted, addr) || len(forwardedFor) == 0 {
		return addr, true
	}
	list := strings.Split(strings.Join(forwardedFor, ","), ",")
	fwd, err := netip.ParseAddr(strings.TrimSpace(list[len(list)-1]))
	if err != nil {
		return netip.Addr{}, false
	}
	return fwd.Unmap(), true
}

// allowed는 addr이 주소 목록(허용 IP, 믿을 proxy) 안에 있는지 알려 준다. 목록이 비어 있으면 false다.
func allowed(allow []netip.Prefix, addr netip.Addr) bool {
	for _, p := range allow {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
