package common

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

var Opts = &Options{
	Bind:         ":8080",
	DB:           "memory://",
	Lock:         "memory://",
	Queue:        "memory://",
	VFS:          "memory://",
	TempDir:      "memory://",
	PubSub:       "memory://",
	Secret:       "plain",
	RootPassword: "admin",
	TotpAuth:     "",
	JWTSecret:    "change-me-please",
	Modules: ModuleOptions{
		Workflow:     true,
		Intelligence: true,
	},
}

type ModuleOptions struct {
	// Workflow 控制是否启用 workflow 模块。
	Workflow bool `yaml:"workflow" env:"HOMELAB_WORKFLOW"`
	// Intelligence 控制是否启用 intelligence 模块。
	Intelligence bool `yaml:"intelligence" env:"HOMELAB_INTELLIGENCE"`
}

type Options struct {
	// Bind 是 HTTP 服务监听地址，例如 ":8080"。
	Bind string `yaml:"bind" env:"HOMELAB_BIND"`
	// DB 是主 KV 存储后端连接串。
	DB string `yaml:"db" env:"HOMELAB_DB"`
	// Lock 是分布式锁后端连接串。
	Lock string `yaml:"lock" env:"HOMELAB_LOCK"`
	// Queue 是异步任务分发队列后端连接串。
	Queue string `yaml:"queue" env:"HOMELAB_QUEUE"`
	// VFS 是持久化用户数据文件系统后端连接串。
	VFS string `yaml:"vfs" env:"HOMELAB_VFS"`
	// TempDir 是任务临时工作目录文件系统后端连接串。
	TempDir string `yaml:"temp_dir" env:"HOMELAB_TEMP_DIR"`
	// PubSub 是集群事件广播订阅后端连接串。
	PubSub string `yaml:"pub_sub" env:"HOMELAB_PUB_SUB"`
	// Secret 是 secret 存储模式，支持 `plain` 或 `aes256:<key>`。
	Secret string `yaml:"secret" env:"HOMELAB_SECRET"`
	// RootPassword 是 root 管理员初始密码。
	RootPassword string `yaml:"password" env:"HOMELAB_PASSWORD"`
	// TotpAuth 是 root 登录使用的 TOTP 二次验证配置。
	TotpAuth string `yaml:"totp_auth" env:"HOMELAB_TOTP_AUTH"`
	// JWTSecret 是会话和服务账号 JWT 的签名密钥。
	JWTSecret string `yaml:"jwt_secret" env:"HOMELAB_JWT_SECRET"`
	// SessionTTL 是登录会话默认存活时长，例如 `30m`。
	SessionTTL string `yaml:"session_ttl" env:"HOMELAB_SESSION_TTL"`
	// Modules 是可选业务模块的启停配置。
	Modules ModuleOptions `yaml:"modules"`
}

func (o *Options) ParseEnv() {
	if o.SessionTTL == "" {
		o.SessionTTL = "30m"
	}
	parseEnvFields(reflect.ValueOf(o).Elem(), "")
}

func (o *Options) ParseEnvWithPrefix(prefix string) {
	parseEnvFields(reflect.ValueOf(o).Elem(), prefix)
}

func parseEnvFields(val reflect.Value, prefix string) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if fieldVal.Kind() == reflect.Struct {
			nextPrefix := prefix
			if nextPrefix != "" {
				tag := field.Tag.Get("yaml")
				if tag == "" {
					tag = strings.ToLower(field.Name)
				}
				nextPrefix = nextPrefix + "_" + strings.ToUpper(strings.ReplaceAll(tag, "-", "_"))
			}
			parseEnvFields(fieldVal, nextPrefix)
			continue
		}

		envKey := field.Tag.Get("env")
		if envKey == "" && prefix != "" {
			tag := field.Tag.Get("yaml")
			if tag == "" {
				tag = strings.ToLower(field.Name)
			}
			envKey = prefix + "_" + strings.ToUpper(strings.ReplaceAll(tag, "-", "_"))
		}
		if envKey == "" {
			continue
		}

		if envVal := os.Getenv(envKey); envVal != "" {
			switch fieldVal.Kind() {
			case reflect.String:
				fieldVal.SetString(envVal)
			case reflect.Bool:
				if parsed, err := strconv.ParseBool(envVal); err == nil {
					fieldVal.SetBool(parsed)
				}
			}
		}
	}
}
