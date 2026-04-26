package dns

import (
	"fmt"
	commonauth "homelab/pkg/common/auth"
	dnsmodel "homelab/pkg/models/network/dns"
	workflowmodel "homelab/pkg/models/workflow"
	authservice "homelab/pkg/services/core/auth"
	actions "homelab/pkg/services/workflow"
)

type DnsRecordProcessor struct{}

func RegisterActionProcessors() {
	actions.Register(&DnsRecordProcessor{})
}

func (p *DnsRecordProcessor) Manifest() actions.StepManifest {
	return actions.StepManifest{
		ID:          "dns/record/create",
		Name:        "DNS Record Creator",
		Description: "在指定域名下创建一条新的解析记录，支持 A, CNAME, TXT 等类型。",
		Params: []workflowmodel.ParamDefinition{
			{Name: "domain_id", Description: "目标域名 ID (如 example.com)", Optional: false, LookupCode: "network/dns/domains"},
			{Name: "name", Description: "主机名 (如 www, @)", Optional: false},
			{Name: "type", Description: "记录类型 (A, CNAME, TXT, etc.)", Optional: false},
			{Name: "value", Description: "记录内容 (IP 地址或目标域名)", Optional: false},
			{Name: "ttl", Description: "生存时间 (秒)，默认 600", Optional: true},
		},
		OutputParams: []workflowmodel.ParamDefinition{
			{Name: "record_id", Description: "新创建的记录唯一 ID"},
		},
	}
}

func (p *DnsRecordProcessor) Execute(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
	domainID := inputs["domain_id"]
	name := inputs["name"]
	recordType := inputs["type"]
	value := inputs["value"]

	ctx.Logger.Logf("Creating DNS record: %s %s -> %s in domain %s", name, recordType, value, domainID)

	record := &dnsmodel.Record{Meta: dnsmodel.RecordV1Meta{
		DomainID: domainID,
		Name:     name,
		Type:     recordType,
		Value:    value,
		Enabled:  true,
	}}

	domain, err := GetDomain(ctx.Context, domainID)
	if err != nil {
		return nil, fmt.Errorf("load domain %s: %w", domainID, err)
	}
	resource := fmt.Sprintf("network/dns/%s/%s/%s", domain.Meta.Name, name, recordType)
	perms, err := authservice.GetPermissions(ctx.Context, ctx.ServiceAccountID, "create", resource)
	if err != nil {
		return nil, fmt.Errorf("load workflow permissions: %w", err)
	}

	res, err := CreateRecord(commonauth.WithPermissions(ctx.Context, perms), record)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"record_id": res.ID,
	}, nil
}
