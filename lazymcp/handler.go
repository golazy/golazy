package lazymcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazyjwt"
)

const protocolVersion = "2025-11-25"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ServeHTTP serves MCP JSON-RPC requests.
func (scope *Scope) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var request rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: err.Error()}})
		return
	}
	result, err := scope.handleRPC(r.Context(), request.Method, request.Params)
	response := rpcResponse{JSONRPC: "2.0", ID: request.ID}
	if err != nil {
		response.Error = &rpcError{Code: -32000, Message: err.Error()}
	} else {
		response.Result = result
	}
	writeRPC(w, response)
}

func (scope *Scope) handleRPC(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{"directoryRead": true},
				"prompts":   map[string]any{},
				"experimental": map[string]any{
					"io.modelcontextprotocol/ui":     map[string]any{},
					"io.modelcontextprotocol/skills": map[string]any{"directoryRead": true},
				},
			},
			"serverInfo": map[string]any{"name": "golazy", "version": "0"},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return scope.listTools(ctx), nil
	case "tools/call":
		var args struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		_ = json.Unmarshal(params, &args)
		return scope.callTool(ctx, args.Name, args.Arguments)
	case "resources/list":
		return scope.listResources(ctx), nil
	case "resources/read":
		var args struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(params, &args)
		return scope.readResource(ctx, args.URI)
	case "resources/templates/list":
		return map[string]any{"resourceTemplates": []any{}}, nil
	case "resources/directory/read":
		var args struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(params, &args)
		return scope.readDirectory(ctx, args.URI)
	case "prompts/list":
		return scope.listPrompts(ctx), nil
	case "prompts/get":
		var args struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		_ = json.Unmarshal(params, &args)
		return scope.getPrompt(ctx, args.Name, args.Arguments)
	default:
		return nil, fmt.Errorf("lazymcp: unsupported method %q", method)
	}
}

func writeRPC(w http.ResponseWriter, response rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (scope *Scope) listTools(ctx context.Context) any {
	var tools []any
	for _, module := range scope.allowedModules(ctx) {
		for name, spec := range module.tools {
			if len(spec.Visibility) > 0 && !contains(spec.Visibility, VisibilityModel) {
				continue
			}
			tool := map[string]any{
				"name":        module.name + "." + name,
				"description": spec.Desc,
				"inputSchema": inputSchema(spec.Fn),
			}
			if spec.UI.ResourceURI != "" {
				tool["_meta"] = map[string]any{"ui/resourceUri": spec.UI.ResourceURI}
			}
			tools = append(tools, tool)
		}
	}
	return map[string]any{"tools": tools}
}

func (scope *Scope) callTool(ctx context.Context, fullName string, arguments json.RawMessage) (any, error) {
	moduleName, name, ok := strings.Cut(fullName, ".")
	if !ok {
		return nil, fmt.Errorf("lazymcp: tool name must include module prefix")
	}
	module, err := scope.authorizedModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	spec, ok := module.tools[name]
	if !ok {
		return nil, fmt.Errorf("lazymcp: tool %q not found", fullName)
	}
	result, err := callFunction(ctx, spec.Fn, arguments)
	if err != nil {
		return nil, err
	}
	text := resultText(result)
	return map[string]any{
		"content":           []any{map[string]any{"type": "text", "text": text}},
		"structuredContent": result,
	}, nil
}

func (scope *Scope) listResources(ctx context.Context) any {
	var resources []any
	for _, module := range scope.allowedModules(ctx) {
		for _, spec := range module.resources {
			resources = append(resources, resourceMetadata(spec.URI, spec.Name, spec.Desc, spec.MIMEType, nil))
		}
		for _, spec := range module.apps {
			meta := appMeta(spec)
			resources = append(resources, resourceMetadata(spec.URI, spec.Name, spec.Desc, "text/html;profile=mcp-app", meta))
		}
	}
	resources = append(resources, scope.skillIndexMetadata(ctx)...)
	return map[string]any{"resources": resources}
}

func (scope *Scope) readResource(ctx context.Context, uri string) (any, error) {
	switch {
	case uri == "skill://index.json":
		return scope.readSkillIndex(ctx)
	case strings.HasPrefix(uri, "skill://"):
		return scope.readSkillResource(ctx, uri)
	case strings.HasPrefix(uri, "ui://"):
		return scope.readApp(ctx, uri)
	default:
		for _, module := range scope.allowedModules(ctx) {
			if spec, ok := module.resources[uri]; ok {
				content, err := spec.Read(ctx)
				if err != nil {
					return nil, err
				}
				return resourceReadResult(uri, firstNonEmpty(content.MIMEType, spec.MIMEType), content), nil
			}
		}
		return nil, fmt.Errorf("lazymcp: resource %q not found", uri)
	}
}

func (scope *Scope) readApp(ctx context.Context, uri string) (any, error) {
	moduleName, _, ok := strings.Cut(strings.TrimPrefix(uri, "ui://"), "/")
	if !ok {
		return nil, fmt.Errorf("lazymcp: invalid app URI %q", uri)
	}
	module, err := scope.authorizedModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	spec, ok := module.apps[uri]
	if !ok {
		return nil, fmt.Errorf("lazymcp: app %q not found", uri)
	}
	html := spec.HTML
	if html == "" {
		if scope.options.Views == nil {
			return nil, fmt.Errorf("lazymcp: views renderer is required for app %q", uri)
		}
		vars := map[string]any{}
		if spec.Variables != nil {
			renderVars, err := spec.Variables(ctx)
			if err != nil {
				return nil, err
			}
			vars = renderVars
		}
		layout := spec.Layout
		if spec.UseLayout && layout == "" {
			layout = "mcp_app"
		}
		rendered, err := scope.options.Views.RenderString(ctx, ViewOptions{
			View:      "mcp/" + module.name + "/" + spec.View,
			Variables: vars,
			Layout:    layout,
			UseLayout: spec.UseLayout,
		})
		if err != nil {
			return nil, err
		}
		html = rendered
	}
	return resourceReadResult(uri, "text/html;profile=mcp-app", ResourceContent{Text: html}), nil
}

func (scope *Scope) listPrompts(ctx context.Context) any {
	var prompts []any
	for _, module := range scope.allowedModules(ctx) {
		for name, spec := range module.prompts {
			prompts = append(prompts, map[string]any{
				"name":        module.name + "." + name,
				"description": spec.Desc,
			})
		}
	}
	return map[string]any{"prompts": prompts}
}

func (scope *Scope) getPrompt(ctx context.Context, fullName string, arguments map[string]string) (any, error) {
	moduleName, name, ok := strings.Cut(fullName, ".")
	if !ok {
		return nil, fmt.Errorf("lazymcp: prompt name must include module prefix")
	}
	module, err := scope.authorizedModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	spec, ok := module.prompts[name]
	if !ok {
		return nil, fmt.Errorf("lazymcp: prompt %q not found", fullName)
	}
	messages := spec.Messages
	if spec.Fn != nil {
		result, err := callFunction(ctx, spec.Fn, mustJSON(arguments))
		if err != nil {
			return nil, err
		}
		if typed, ok := result.([]Message); ok {
			messages = typed
		}
	}
	out := make([]any, 0, len(messages))
	for _, message := range messages {
		out = append(out, map[string]any{
			"role": message.Role,
			"content": map[string]any{
				"type": "text",
				"text": message.Text,
			},
		})
	}
	return map[string]any{"messages": out}, nil
}

func (scope *Scope) allowedModules(ctx context.Context) []*module {
	modules := make([]*module, 0, len(scope.modules))
	for name, module := range scope.modules {
		if scope.allowed(ctx, name) {
			modules = append(modules, module)
		}
	}
	return modules
}

func (scope *Scope) authorizedModule(ctx context.Context, name string) (*module, error) {
	module, ok := scope.modules[name]
	if !ok {
		return nil, fmt.Errorf("lazymcp: module %q not found", name)
	}
	if !scope.allowed(ctx, name) {
		return nil, fmt.Errorf("lazymcp: module %q is not authorized", name)
	}
	return module, nil
}

func (scope *Scope) allowed(ctx context.Context, moduleName string) bool {
	if scope.options.Authorizer != nil {
		return scope.options.Authorizer(ctx, moduleName)
	}
	claims, ok := lazyjwt.ClaimsFromContext(ctx)
	if !ok {
		return true
	}
	mcps := claims.StringSlice("mcps")
	if len(mcps) == 0 {
		return false
	}
	return contains(mcps, moduleName) || contains(mcps, "*")
}

func resourceMetadata(uri string, name string, desc string, mimeType string, meta map[string]any) map[string]any {
	out := map[string]any{
		"uri":         uri,
		"name":        name,
		"description": desc,
		"mimeType":    mimeType,
	}
	if meta != nil {
		out["_meta"] = meta
	}
	return out
}

func resourceReadResult(uri string, mimeType string, content ResourceContent) map[string]any {
	item := map[string]any{"uri": uri, "mimeType": mimeType}
	if len(content.Blob) > 0 {
		item["blob"] = base64.StdEncoding.EncodeToString(content.Blob)
	} else {
		item["text"] = content.Text
	}
	return map[string]any{"contents": []any{item}}
}

func appMeta(spec AppSpec) map[string]any {
	return map[string]any{
		"ui.name":          spec.Name,
		"ui.description":   spec.Desc,
		"ui.csp":           spec.CSP,
		"ui.permissions":   spec.Permissions,
		"ui.domain":        spec.Domain,
		"ui.prefersBorder": spec.PrefersBorder,
	}
}

func callFunction(ctx context.Context, fn any, raw json.RawMessage) (any, error) {
	if fn == nil {
		return nil, fmt.Errorf("lazymcp: function is nil")
	}
	value := reflect.ValueOf(fn)
	typ := value.Type()
	args := []reflect.Value{}
	argIndex := 0
	if typ.NumIn() > 0 && typ.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() {
		args = append(args, reflect.ValueOf(ctx))
		argIndex = 1
	}
	if typ.NumIn() > argIndex {
		argType := typ.In(argIndex)
		arg := reflect.New(argType)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, arg.Interface()); err != nil {
				return nil, err
			}
		}
		args = append(args, arg.Elem())
	}
	out := value.Call(args)
	if len(out) == 0 {
		return nil, nil
	}
	if len(out) > 1 && !out[len(out)-1].IsNil() {
		err, _ := out[len(out)-1].Interface().(error)
		return nil, err
	}
	return out[0].Interface(), nil
}

func inputSchema(fn any) map[string]any {
	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	if fn == nil {
		return schema
	}
	typ := reflect.TypeOf(fn)
	input := -1
	for i := 0; i < typ.NumIn(); i++ {
		if typ.In(i) == reflect.TypeOf((*context.Context)(nil)).Elem() {
			continue
		}
		input = i
		break
	}
	if input < 0 || typ.In(input).Kind() != reflect.Struct {
		return schema
	}
	properties := map[string]any{}
	for i := 0; i < typ.In(input).NumField(); i++ {
		field := typ.In(input).Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" {
			name = camelToSnake(field.Name)
		}
		if name == "-" {
			continue
		}
		properties[name] = map[string]any{"type": jsonSchemaType(field.Type)}
	}
	schema["properties"] = properties
	return schema
}

func jsonSchemaType(typ reflect.Type) string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "string"
	}
}

func resultText(result any) string {
	if result == nil {
		return ""
	}
	if text, ok := result.(string); ok {
		return text
	}
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprint(result)
	}
	return string(data)
}

func mustJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func minifiedJSON(value any) string {
	data, _ := json.Marshal(value)
	var out bytes.Buffer
	_ = json.Compact(&out, data)
	return out.String()
}
