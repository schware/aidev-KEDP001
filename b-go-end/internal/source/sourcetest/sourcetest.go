// Package sourcetest는 제공처 테스트가 함께 쓰는 도구다.
// 받아 둔 샘플만 읽는다. 실제 API는 부르지 않는다(KEDP001-D021).
package sourcetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SampleDir는 제공처 패키지(internal/source/<제공처>)에서 본 samples/<dir> 위치다.
func SampleDir(dir string) string {
	return filepath.Join("..", "..", "..", "..", "samples", dir)
}

// ReadSample은 dir 안의 샘플 파일을 읽는다.
func ReadSample(t *testing.T, dir, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// CheckMasked는 dir 안의 샘플마다 marker(키 바로 앞 문자열)가 나오면 그 뒤가 ***인지 확인한다.
func CheckMasked(t *testing.T, dir, marker string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b := string(ReadSample(t, dir, e.Name()))
		for i := strings.Index(b, marker); i >= 0; {
			if !strings.HasPrefix(b[i+len(marker):], "***") {
				t.Errorf("%s: %q without masked key", e.Name(), marker)
				break
			}
			next := strings.Index(b[i+len(marker):], marker)
			if next < 0 {
				break
			}
			i += len(marker) + next
		}
	}
}
