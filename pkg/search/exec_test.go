package search_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

// shCmd creates an ExecProvider that runs /bin/sh -c "script body".
// This avoids temp script files and the "text file busy" race on Linux/overlayfs.
func shCmd(t *testing.T, name string, engines []string, script string) *search.ExecProvider {
	t.Helper()
	return search.NewExecProvider(name, "/bin/sh", engines).WithArgs("-c", script)
}

func TestExecProvider_Name(t *testing.T) {
	t.Parallel()
	p := search.NewExecProvider("myplugin", "/bin/true", []string{"web"})
	assert.Equal(t, "myplugin", p.Name())
}

func TestExecProvider_Engines(t *testing.T) {
	t.Parallel()
	engines := []string{"web", "images", "news"}
	p := search.NewExecProvider("test", "/bin/true", engines)
	assert.Equal(t, engines, p.Engines())
}

func TestExecProvider_Search_Success(t *testing.T) {
	t.Parallel()

	p := shCmd(t, "test", []string{"custom"},
		`cat <<'EOF'
{"query":"test","engine":"custom","provider":"test","results":[{"position":1,"title":"Exec Result","url":"https://example.com","snippet":"from plugin"}]}
EOF`)

	resp, err := p.Search(context.Background(), "test", search.Options{Engine: "custom"})
	require.NoError(t, err)

	assert.Equal(t, "test", resp.Provider)
	assert.Equal(t, "test", resp.Query)
	assert.Equal(t, "custom", resp.Engine)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "Exec Result", resp.Results[0].Title)
	assert.Equal(t, "https://example.com", resp.Results[0].URL)
	assert.Equal(t, "from plugin", resp.Results[0].Snippet)
	assert.Equal(t, 1, resp.Results[0].Position)
}

func TestExecProvider_Search_PassesInput(t *testing.T) {
	t.Parallel()

	p := shCmd(t, "input-test", []string{"echo"},
		`INPUT=$(cat)
cat <<EOF
{"query":"check","engine":"echo","provider":"input-test","results":[],"metadata":{"stdin":${INPUT}}}
EOF`)

	opts := search.Options{
		Engine:  "echo",
		Lang:    "en",
		Country: "us",
		Limit:   5,
		Page:    2,
		Raw:     true,
	}
	resp, err := p.Search(context.Background(), "hello world", opts)
	require.NoError(t, err)

	raw, ok := resp.Metadata["stdin"].(map[string]any)
	require.True(t, ok, "metadata.stdin should be a JSON object")

	assert.Equal(t, "hello world", raw["query"])
	assert.Equal(t, "echo", raw["engine"])

	optMap, ok := raw["options"].(map[string]any)
	require.True(t, ok, "options should be a JSON object")

	assert.Equal(t, "en", optMap["lang"])
	assert.Equal(t, "us", optMap["country"])
	assert.EqualValues(t, 5, optMap["limit"])
	assert.EqualValues(t, 2, optMap["page"])
	assert.Equal(t, true, optMap["raw"])
}

func TestExecProvider_Search_NonZeroExit(t *testing.T) {
	t.Parallel()
	p := shCmd(t, "fail", []string{"web"}, "exit 1")

	_, err := p.Search(context.Background(), "test", search.Options{Engine: "web"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run command")
}

func TestExecProvider_Search_InvalidJSON(t *testing.T) {
	t.Parallel()
	p := shCmd(t, "badjson", []string{"web"}, `echo "not json"`)

	_, err := p.Search(context.Background(), "test", search.Options{Engine: "web"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestExecProvider_Search_Timeout(t *testing.T) {
	t.Parallel()
	p := shCmd(t, "slow", []string{"web"}, "sleep 10").
		WithTimeout(100 * time.Millisecond)

	_, err := p.Search(context.Background(), "test", search.Options{Engine: "web"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run command")
}

func TestExecProvider_Search_MissingBinary(t *testing.T) {
	t.Parallel()
	p := search.NewExecProvider("missing", "/nonexistent/binary", []string{"web"})

	_, err := p.Search(context.Background(), "test", search.Options{Engine: "web"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run command")
}

func TestExecProvider_Search_EmptyOutput(t *testing.T) {
	t.Parallel()
	p := shCmd(t, "empty", []string{"web"}, "exit 0")

	_, err := p.Search(context.Background(), "test", search.Options{Engine: "web"})
	require.Error(t, err)
	assert.ErrorIs(t, err, search.ErrExecEmptyOutput)
}

// Verify ExecProvider satisfies the Provider interface at compile time.
var _ search.Provider = (*search.ExecProvider)(nil)

func TestExecProvider_RequestFormat(t *testing.T) {
	t.Parallel()

	type execRequest struct {
		Query   string `json:"query"`
		Engine  string `json:"engine"`
		Options struct {
			Lang    string `json:"lang"`
			Country string `json:"country"`
			Limit   int    `json:"limit"`
			Page    int    `json:"page"`
			Raw     bool   `json:"raw"`
		} `json:"options"`
	}

	req := execRequest{Query: "test", Engine: "web"}
	req.Options.Lang = "en"
	req.Options.Limit = 10

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "test", parsed["query"])
	assert.Equal(t, "web", parsed["engine"])
	assert.Contains(t, parsed, "options")
}
