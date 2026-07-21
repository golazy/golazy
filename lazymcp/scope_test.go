package lazymcp

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyjwt"
)

type adminMCP struct {
	Base
}

type userCountParams struct {
	Active bool `json:"active"`
}

type userCountResult struct {
	Count int `json:"count"`
}

func newAdminMCP(ctx context.Context) *adminMCP {
	return &adminMCP{Base: NewBase(ctx)}
}

func (m *adminMCP) UserCountTool(context.Context) ToolSpec {
	return ToolSpec{
		Desc: "Count users.",
		Fn:   m.UserCount,
		UI:   UI("ui://admin/dashboard"),
	}
}

func (m *adminMCP) UserCount(context.Context, userCountParams) (userCountResult, error) {
	return userCountResult{Count: 3}, nil
}

func (m *adminMCP) DashboardApp(context.Context) AppSpec {
	return AppSpec{
		Name:   "dashboard",
		Desc:   "Admin dashboard",
		View:   "dashboard",
		Domain: "https://widgets.example.com",
		CSP: AppCSP{
			ConnectDomains:  []string{"https://api.example.com"},
			ResourceDomains: []string{"https://static.example.com"},
		},
	}
}

func (m *adminMCP) OncallSkill(context.Context) SkillSpec {
	return SkillSpec{
		Path: "oncall",
		FS: fstest.MapFS{
			"SKILL.md":              {Data: []byte("---\nname: oncall\ndescription: Handle on-call work\n---\nUse the runbook.")},
			"references/runbook.md": {Data: []byte("Runbook")},
		},
	}
}

type fakeViews struct {
	view string
}

func (views *fakeViews) RenderString(_ context.Context, options ViewOptions) (string, error) {
	views.view = options.View
	return "<!doctype html><p>dashboard</p>", nil
}

func TestScopeRegistersReflectedToolsAppsAndSkills(t *testing.T) {
	views := &fakeViews{}
	scope := NewScope(Options{Views: views})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	claims := lazyjwt.Claims{Extra: map[string]any{"mcps": []string{"admin"}}}
	ctx := lazyjwt.WithClaims(context.Background(), claims)
	tools := scope.listTools(ctx).(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(tools))
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "admin.user_count" {
		t.Fatalf("tool name = %q", tool["name"])
	}
	meta := tool["_meta"].(map[string]any)
	if meta["ui/resourceUri"] != "ui://admin/dashboard" {
		t.Fatalf("tool ui resource = %q", meta["ui/resourceUri"])
	}
	if meta["openai/outputTemplate"] != "ui://admin/dashboard" {
		t.Fatalf("tool output template = %q", meta["openai/outputTemplate"])
	}
	output := tool["outputSchema"].(map[string]any)
	if output["type"] != "object" {
		t.Fatalf("tool output schema = %#v", output)
	}
	properties := output["properties"].(map[string]any)
	if properties["count"].(map[string]any)["type"] != "integer" {
		t.Fatalf("count output schema = %#v", properties["count"])
	}
	appResult, err := scope.readApp(ctx, "ui://admin/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	appContent := appResult.(map[string]any)["contents"].([]any)[0].(map[string]any)
	appMetadata := appContent["_meta"].(map[string]any)
	ui := appMetadata["ui"].(map[string]any)
	if ui["domain"] != "https://widgets.example.com" {
		t.Fatalf("app domain = %#v", ui["domain"])
	}
	csp := ui["csp"].(map[string]any)
	if !slices.Equal(csp["connectDomains"].([]string), []string{"https://api.example.com"}) {
		t.Fatalf("app CSP = %#v", csp)
	}
	if appMetadata["openai/widgetDomain"] != "https://widgets.example.com" {
		t.Fatalf("legacy widget domain = %#v", appMetadata["openai/widgetDomain"])
	}
	if views.view != "mcp/admin/dashboard" {
		t.Fatalf("rendered view = %q, want mcp/admin/dashboard", views.view)
	}
	index, err := scope.readSkillIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	body := index.(map[string]any)["contents"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(body, "skill://admin/oncall/SKILL.md") {
		t.Fatalf("skill index = %s", body)
	}
}

func TestScopeDeniesUnauthorizedModule(t *testing.T) {
	scope := NewScope(Options{})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	ctx := lazyjwt.WithClaims(context.Background(), lazyjwt.Claims{Extra: map[string]any{"mcps": []string{"tenant"}}})
	if _, err := scope.callTool(ctx, "admin.user_count", nil); err == nil {
		t.Fatal("callTool succeeded without admin grant")
	}
}

func TestScopeListsProtocolArraysWhenNoModuleIsAuthorized(t *testing.T) {
	scope := NewScope(Options{})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	ctx := lazyjwt.WithClaims(context.Background(), lazyjwt.Claims{Extra: map[string]any{"mcps": []string{"tenant"}}})
	for name, result := range map[string]map[string]any{
		"tools":     scope.listTools(ctx).(map[string]any),
		"resources": scope.listResources(ctx).(map[string]any),
		"prompts":   scope.listPrompts(ctx).(map[string]any),
	} {
		items, ok := result[name].([]any)
		if !ok {
			t.Fatalf("%s = %#v, want an array", name, result[name])
		}
		if len(items) != 0 {
			t.Fatalf("%s length = %d, want 0", name, len(items))
		}
	}
}

func TestScopeServesJSONRPC(t *testing.T) {
	scope := NewScope(Options{})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"admin.user_count","arguments":{"active":true}}}`
	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var decoded struct {
		Result struct {
			Structured userCountResult `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Result.Structured.Count != 3 {
		t.Fatalf("count = %d, want 3", decoded.Result.Structured.Count)
	}
}

func TestSkillReadAndDirectory(t *testing.T) {
	scope := NewScope(Options{})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	result, err := scope.readSkillResource(context.Background(), "skill://admin/oncall/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.(map[string]any)["contents"].([]any)[0].(map[string]any)["text"].(string), "Use the runbook") {
		t.Fatal("skill body missing")
	}
	dir, err := scope.readDirectory(context.Background(), "skill://admin/oncall/references")
	if err != nil && !strings.Contains(err.Error(), fs.ErrNotExist.Error()) {
		t.Fatal(err)
	}
	if err == nil && len(dir.(map[string]any)["entries"].([]any)) == 0 {
		t.Fatal("directory was empty")
	}
}
