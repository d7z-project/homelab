package workflow

func GenerateWorkflowSchema() map[string]interface{} {
	manifests := ScanManifests()
	stepDefinitions := make(map[string]interface{})

	for _, m := range manifests {
		if m.ID == "" {
			continue
		}

		paramProps := make(map[string]interface{})
		var requiredParams []string
		if m.Params != nil {
			for _, p := range m.Params {
				if p.Name != "" {
					paramProps[p.Name] = map[string]interface{}{
						"type":        "string",
						"description": p.Description,
					}
					if !p.Optional {
						requiredParams = append(requiredParams, p.Name)
					}
				}
			}
		}

		stepDefinitions[m.ID] = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id":   map[string]interface{}{"type": "string", "pattern": "^[a-z0-9_]+$"},
				"name": map[string]interface{}{"type": "string"},
				"type": map[string]interface{}{"const": m.ID},
				"if":   map[string]interface{}{"type": "string"},
				"params": map[string]interface{}{
					"type":       "object",
					"properties": paramProps,
					"required":   requiredParams,
				},
				"fail": map[string]interface{}{
					"type":        "boolean",
					"description": "执行出错时是否继续执行后续步骤",
					"default":     false,
				},
			},
			"required": []string{"id", "type"},
		}
	}

	anyOf := make([]interface{}, 0, len(stepDefinitions))
	for _, def := range stepDefinitions {
		anyOf = append(anyOf, def)
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":               map[string]interface{}{"type": "string", "description": "工作流唯一标识"},
			"name":             map[string]interface{}{"type": "string", "description": "工作流名称"},
			"description":      map[string]interface{}{"type": "string", "description": "工作流描述"},
			"enabled":          map[string]interface{}{"type": "boolean", "description": "是否启用", "default": true},
			"timeout":          map[string]interface{}{"type": "integer", "description": "超时时间(秒)", "default": 7200},
			"serviceAccountId": map[string]interface{}{"type": "string", "description": "执行身份"},
			"cronEnabled":      map[string]interface{}{"type": "boolean", "description": "是否开启定时任务", "default": false},
			"cronExpr":         map[string]interface{}{"type": "string", "description": "Cron 表达式"},
			"webhookEnabled":   map[string]interface{}{"type": "boolean", "description": "是否开启 Webhook", "default": false},
			"vars": map[string]interface{}{
				"type": "object",
				"additionalProperties": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"description":   map[string]interface{}{"type": "string"},
						"default":       map[string]interface{}{"type": "string"},
						"required":      map[string]interface{}{"type": "boolean", "default": false},
						"regexFrontend": map[string]interface{}{"type": "string"},
						"regexBackend":  map[string]interface{}{"type": "string"},
					},
				},
			},
			"steps": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"anyOf": anyOf},
			},
		},
		"required": []string{"name", "serviceAccountId", "steps"},
	}
}
