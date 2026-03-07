package ip

import (
	"crypto/rand"
	"fmt"
	"homelab/pkg/models"
	"net"
	"net/url"
)

func generatePolicyID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 10)
	b[0] = '_'
	rb := make([]byte, 9)
	_, _ = rand.Read(rb)
	for i := 0; i < 9; i++ {
		b[i+1] = letters[rb[i]%uint8(len(letters))]
	}
	return string(b)
}

func validateSourceURL(urlStr string, policy *models.IPSyncPolicy) error {
	allowPrivate := false
	if policy != nil && policy.Config != nil && policy.Config["allowPrivate"] == "true" {
		allowPrivate = true
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := u.Hostname()
	if hostname == "localhost" && !allowPrivate {
		return fmt.Errorf("SSRF detected: localhost is forbidden")
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		// 如果解析失败，可能是 IP 直连或者非法的
		ip := net.ParseIP(hostname)
		if ip != nil {
			if isPrivateIP(ip) && !allowPrivate {
				return fmt.Errorf("SSRF detected: private IP %s is forbidden", ip)
			}
			return nil
		}
		// 暂时允许无法解析的情况（如容器内 DNS），但在生产中应更严格
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) && !allowPrivate {
			return fmt.Errorf("SSRF detected: host %s resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}
