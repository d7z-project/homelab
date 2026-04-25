package processors

import (
	"fmt"
	"homelab/pkg/common"
	"io"
	"net/http"

	workflowmodel "homelab/pkg/models/workflow"
	actions "homelab/pkg/services/workflow"
)

type HttpFetchProcessor struct{}

func RegisterHTTPProcessors() {
	actions.Register(&HttpFetchProcessor{})
}

func (p *HttpFetchProcessor) Manifest() actions.StepManifest {
	return actions.StepManifest{
		ID:          "core/fetch/http",
		Name:        "HTTP Fetcher",
		Description: "从指定 URL 下载文件到任务工作目录，支持 HTTP/HTTPS 协议。",
		Params: []workflowmodel.ParamDefinition{
			{
				Name:          "url",
				Description:   "要下载的远程文件 URL 地址",
				Optional:      false,
				RegexFrontend: `^https?://.+`,
				RegexBackend:  `^https?://.+`,
			},
			{Name: "output_file", Description: "本地保存的文件名，默认为 downloaded_file", Optional: true},
		},
		OutputParams: []workflowmodel.ParamDefinition{
			{Name: "file_path", Description: "下载成功后文件在工作目录的绝对路径"},
		},
	}
}

func (p *HttpFetchProcessor) Execute(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
	url, ok := inputs["url"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: url")
	}

	outputFile := inputs["output_file"]
	if outputFile == "" {
		outputFile = "downloaded_file"
	}
	if err := common.ValidateURL(url, false); err != nil {
		return nil, err
	}

	ctx.Logger.Logf("Fetching URL: %s to workspace file: %s", url, outputFile)

	req, err := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	out, err := ctx.Workspace.Create(outputFile)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"file_name": outputFile,
	}, nil
}
