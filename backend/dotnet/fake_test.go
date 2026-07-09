package dotnet

import (
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// fakeRunner 是 CommandRunner 的测试替身，行为由函数字段决定。
type fakeRunner struct {
	lookPathFn func(string) error
	runFn      func(name string, args []string) (stdout, stderr string, code int, err error)
	calls      []fakeRun
}

// fakeRun 记录一次 Run 调用。
type fakeRun struct {
	Name string
	Args []string
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if f.lookPathFn != nil {
		if err := f.lookPathFn(file); err != nil {
			return "", err
		}
	}
	return "/fake/" + file, nil
}

func (f *fakeRunner) Run(name string, args ...string) (string, string, int, error) {
	f.calls = append(f.calls, fakeRun{Name: name, Args: append([]string(nil), args...)})
	if f.runFn != nil {
		return f.runFn(name, args)
	}
	return "", "", 0, nil
}

// sha512Hex 计算字节的 sha512 十六进制摘要。
func sha512Hex(b []byte) string {
	sum := sha512.Sum512(b)
	return hex.EncodeToString(sum[:])
}

// writeFile 在 dir 下写入一个文件，失败时终止测试。
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
