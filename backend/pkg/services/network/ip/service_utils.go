package ip

import (
	"crypto/rand"
	"homelab/pkg/common"
	ipmodel "homelab/pkg/models/network/ip"
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

func validateSourceURL(urlStr string, policy *ipmodel.IPSyncPolicy) error {
	allowPrivate := false
	if policy != nil && policy.Meta.Config != nil && policy.Meta.Config["allowPrivate"] == "true" {
		allowPrivate = true
	}
	return common.ValidateURL(urlStr, allowPrivate)
}
