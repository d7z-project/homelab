package rules

import networkcommon "homelab/pkg/models/network/common"

type IPEnricher interface {
	Lookup(ipStr string) (*networkcommon.IPInfoResponse, error)
}
