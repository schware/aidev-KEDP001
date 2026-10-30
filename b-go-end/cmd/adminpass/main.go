// adminpass는 관리자 계정 파일(KEDP_ADMIN_FILE)에 넣을 한 줄("계정:bcrypt hash")을 만든다(KEDP001-D018).
//
//	go run ./cmd/adminpass -account <계정>
//
// 비밀번호는 화면에 보이지 않게 두 번 입력받는다(12자 이상). 출력은 파일에 넣을 한 줄뿐이다.
// 비밀번호 원문은 어디에도 저장하지 않는다. 파일은 소유자가 만든다.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/admin"
)

func main() {
	account := flag.String("account", "", "관리자 계정 이름(영문, 숫자, '.', '_', '-', 1~64자)")
	flag.Parse()

	if err := run(*account, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "adminpass:", err)
		os.Exit(1)
	}
}

func run(account string, out, prompt io.Writer) error {
	if !admin.ValidAccount(account) {
		return errors.New("-account must be 1-64 characters of A-Z, a-z, 0-9, '.', '_', '-'")
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("run in a terminal: the password is read without echo")
	}
	read := func(label string) (string, error) {
		fmt.Fprint(prompt, label)
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(prompt)
		return string(b), err
	}
	pw, err := read("password: ")
	if err != nil {
		return err
	}
	if err := admin.CheckPassword(pw); err != nil {
		return err
	}
	again, err := read("password again: ")
	if err != nil {
		return err
	}
	if pw != again {
		return errors.New("passwords do not match")
	}
	line, err := admin.HashLine(account, pw)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(out)
	fmt.Fprintln(w, line)
	return w.Flush()
}
