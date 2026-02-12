package auth

import (
	"fmt"
	"testing"

	"github.com/pquerna/otp/totp"
)

// TestGenerateTotpSecret 演示如何生成一个新的 TOTP 密钥
// 运行命令: cd backend/pkg/auth && go test -v totp_gen_test.go auth.go
func TestGenerateTotpSecret(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Homelab",
		AccountName: "root",
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("==================================================")
	fmt.Printf("1. 将此密钥填入 config.yaml 的 totp_auth 字段:\n   %s\n\n", key.Secret())
	fmt.Printf("2. 在手机 App 中手动输入密钥，或使用以下链接生成二维码扫描:\n   %s\n", key.URL())
	fmt.Println("==================================================")
}
