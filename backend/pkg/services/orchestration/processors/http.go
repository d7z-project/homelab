package processors

import (
	"fmt"
	"homelab/pkg/services/orchestration"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type HttpFetchProcessor struct{}

func init() {
	orchestration.Register(&HttpFetchProcessor{})
}

func (p *HttpFetchProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:             "core/fetch/http",
		Name:           "HTTP Fetcher",
		RequiredParams: []string{"url"},
		OptionalParams: []string{"output_file"},
		OutputParams:   []string{"file_path"},
	}
}

func (p *HttpFetchProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	url, ok := inputs["url"]
	if !ok {
		return nil, fmt.Errorf("missing required parameter: url")
	}

	outputFile := inputs["output_file"]
	if outputFile == "" {
		outputFile = "downloaded_file"
	}
	filePath := filepath.Join(ctx.Workspace, outputFile)

	ctx.Logger.Logf("Fetching URL: %s to %s", url, filePath)

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

	out, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"file_path": filePath,
	}, nil
}
