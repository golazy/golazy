package lazymcp

import (
	"context"
	"io/fs"
)

const (
	VisibilityModel = "model"
	VisibilityApp   = "app"
)

// ToolSpec describes one MCP tool.
type ToolSpec struct {
	Name       string
	Desc       string
	Fn         any
	UI         UIRef
	Visibility []string
}

// UIRef links a tool to an MCP Apps UI resource.
type UIRef struct {
	ResourceURI string
}

// UI returns a UIRef for uri.
func UI(uri string) UIRef {
	return UIRef{ResourceURI: uri}
}

// ResourceSpec describes one MCP resource.
type ResourceSpec struct {
	URI      string
	Name     string
	Desc     string
	MIMEType string
	Read     func(context.Context) (ResourceContent, error)
}

// ResourceContent is returned by ResourceSpec readers.
type ResourceContent struct {
	Text     string
	Blob     []byte
	MIMEType string
}

// PromptSpec describes one MCP prompt.
type PromptSpec struct {
	Name     string
	Desc     string
	Messages []Message
	Fn       any
}

// Message is an MCP prompt message.
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// UserMessages creates user-role prompt messages.
func UserMessages(messages ...string) []Message {
	out := make([]Message, 0, len(messages))
	for _, message := range messages {
		out = append(out, Message{Role: "user", Text: message})
	}
	return out
}

// SkillSpec describes an Agent Skill directory served through MCP resources.
type SkillSpec struct {
	Path string
	FS   fs.FS
}

// AppSpec describes one MCP Apps UI resource owned by an MCP module.
type AppSpec struct {
	URI  string
	Name string
	Desc string

	HTML      string
	View      string
	Variables func(context.Context) (map[string]any, error)
	Layout    string
	UseLayout bool

	CSP           AppCSP
	Permissions   AppPermissions
	Domain        string
	PrefersBorder bool
}

// AppCSP describes MCP Apps content security policy metadata.
type AppCSP struct {
	ConnectDomains  []string `json:"connect_domains,omitempty"`
	ResourceDomains []string `json:"resource_domains,omitempty"`
}

// AppPermissions describes optional MCP Apps permission metadata.
type AppPermissions struct {
	Tools []string `json:"tools,omitempty"`
}

// ViewOptions configures rendering for one MCP App view.
type ViewOptions struct {
	View      string
	Variables map[string]any
	Layout    string
	UseLayout bool
}

// Views renders MCP App views. lazyapp adapts lazyview.Views to this interface.
type Views interface {
	RenderString(context.Context, ViewOptions) (string, error)
}
