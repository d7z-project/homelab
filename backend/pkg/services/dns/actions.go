package dns

import (
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
)

type DnsRecordProcessor struct{}

func init() {
	actions.Register(&DnsRecordProcessor{})
}

func (p *DnsRecordProcessor) Manifest() actions.StepManifest {
	return actions.StepManifest{
		ID:          "dns/record/create",
		Name:        "DNS Record Creator",
		Description: "在指定域名下创建一条新的解析记录，支持 A, CNAME, TXT 等类型。",
		Params: []models.ParamDefinition{
			{Name: "domain_id", Description: "目标域名 ID (如 example.com)", Optional: false, LookupCode: "network/dns/domains"},
			{Name: "name", Description: "主机名 (如 www, @)", Optional: false},
			{Name: "type", Description: "记录类型 (A, CNAME, TXT, etc.)", Optional: false},
			{Name: "value", Description: "记录内容 (IP 地址或目标域名)", Optional: false},
			{Name: "ttl", Description: "生存时间 (秒)，默认 600", Optional: true},
		},
		OutputParams: []models.ParamDefinition{
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

	// Simulated Live RBAC Check
	if ctx.UserID != "root" && ctx.UserID != "admin" {
		return nil, fmt.Errorf("user %s does not have permission to modify DNS", ctx.UserID)
	}

	record := &models.Record{Meta: models.RecordV1Meta{
		DomainID: domainID,
		Name:     name,
		Type:     recordType,
		Value:    value,
		Enabled:  true,
	}}

	// Create a context with full permissions for the internal service call
	// The service layer itself will perform the final validation
	adminCtx := commonauth.WithPermissions(ctx.Context, &models.ResourcePermissions{AllowedAll: true})

	res, err := CreateRecord(adminCtx, record)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"record_id": res.ID,
	}, nil
}
