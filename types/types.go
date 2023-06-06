package types

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mitchellh/mapstructure"
)

type PageType string

const (
	DetailPage PageType = "detail"
	ListPage   PageType = "list"
)

type Page struct {
	Type    PageType `json:"type"`
	Title   string   `json:"title,omitempty"`
	Actions []Action `json:"actions,omitempty"`

	// Detail page
	Preview *TextOrCommandOrRequest `json:"preview,omitempty"`

	// List page
	ShowPreview   bool       `json:"showPreview,omitempty"`
	OnQueryChange *Command   `json:"onQueryChange,omitempty"`
	EmptyView     *EmptyView `json:"emptyView,omitempty"`
	Items         []ListItem `json:"items,omitempty"`
}

type EmptyView struct {
	Text    string   `json:"text,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

type ListItem struct {
	Id          string                  `json:"id,omitempty"`
	Title       string                  `json:"title"`
	Subtitle    string                  `json:"subtitle,omitempty"`
	Preview     *TextOrCommandOrRequest `json:"preview,omitempty"`
	Accessories []string                `json:"accessories,omitempty"`
	Actions     []Action                `json:"actions,omitempty"`
}

type FormInputType string

const (
	TextFieldInput FormInputType = "textfield"
	TextAreaInput  FormInputType = "textarea"
	DropDownInput  FormInputType = "dropdown"
	CheckboxInput  FormInputType = "checkbox"
)

type DropDownItem struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type Input struct {
	Name        string        `json:"name"`
	Type        FormInputType `json:"type"`
	Title       string        `json:"title"`
	Placeholder string        `json:"placeholder,omitempty"`
	Default     any           `json:"default,omitempty"`
	Optional    bool          `json:"optional,omitempty"`

	// Only for dropdown
	Items []DropDownItem `json:"items,omitempty"`

	// Only for checkbox
	Label             string `json:"label,omitempty"`
	TrueSubstitution  string `json:"trueSubstitution,omitempty"`
	FalseSubstitution string `json:"falseSubstitution,omitempty"`
}

func NewTextInput(name string, title string, placeholder string) Input {
	return Input{
		Name:        name,
		Type:        TextFieldInput,
		Title:       title,
		Placeholder: placeholder,
	}
}

func NewTextAreaInput(name string, title string, placeholder string) Input {
	return Input{
		Name:        name,
		Type:        TextAreaInput,
		Title:       title,
		Placeholder: placeholder,
	}
}

func NewCheckbox(name string, title string, label string) Input {
	return Input{
		Name:  name,
		Type:  CheckboxInput,
		Title: title,
		Label: label,
	}
}

func NewDropDown(name string, title string, items ...DropDownItem) Input {
	return Input{
		Name:  name,
		Type:  DropDownInput,
		Title: title,
		Items: items,
	}
}

type ActionType string

const (
	CopyAction   = "copy"
	OpenAction   = "open"
	PushAction   = "push"
	RunAction    = "run"
	ExitAction   = "exit"
	PasteAction  = "paste"
	ReloadAction = "reload"
	FetchAction  = "fetch"
)

type Action struct {
	Title  string     `json:"title,omitempty"`
	Type   ActionType `json:"type"`
	Key    string     `json:"key,omitempty"`
	Inputs []Input    `json:"inputs,omitempty"`

	// copy
	Text string `json:"text,omitempty"`

	// open
	Target string `json:"target,omitempty"`

	// push
	Page *PageGenerator `json:"page,omitempty"`

	// fetch
	Request *Request `json:"request,omitempty"`

	// run
	Command         *Command `json:"command,omitempty"`
	ReloadOnSuccess bool     `json:"reloadOnSuccess,omitempty"`
}

type PageGenerator struct {
	Command *Command `json:"command,omitempty"`
	Request *Request `json:"request,omitempty"`
	Path    string   `json:"path,omitempty"`
}

func (p *PageGenerator) UnmarshalJSON(data []byte) error {
	var c Command
	if err := json.Unmarshal(data, &c); err == nil && c.Name != "" {
		p.Command = &c
		return nil
	}

	var r Request
	if err := json.Unmarshal(data, &r); err == nil && r.Url != "" {
		p.Request = &r
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		p.Path = s
		return nil
	}

	return errors.New("must be a command or a request")
}

func (p PageGenerator) MarshalJSON() ([]byte, error) {
	if p.Command != nil {
		return json.Marshal(p.Command)
	}

	if p.Request != nil {
		return json.Marshal(p.Request)
	}

	if p.Path != "" {
		return json.Marshal(p.Path)
	}

	return nil, errors.New("must be a command or a request")
}

type Request struct {
	Url     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    Body              `json:"body,omitempty"`
}

func (r Request) Do() ([]byte, error) {
	if r.Method == "" {
		r.Method = "GET"
	}

	req, err := http.NewRequest(r.Method, r.Url, bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}

	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.New(resp.Status)
	}

	return io.ReadAll(resp.Body)
}

type Body []byte

func (b *Body) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*b = Body(s)
		return nil
	}

	var v map[string]any
	if err := json.Unmarshal(data, &v); err == nil {
		bb, err := json.Marshal(v)
		if err != nil {
			return err
		}
		*b = Body(bb)
		return nil
	}

	return errors.New("body must be a string or a map")
}

type TextOrCommandOrRequest struct {
	Text    string   `json:"text,omitempty"`
	Command *Command `json:"command,omitempty"`
	Request *Request `json:"request,omitempty"`
}

func (p *TextOrCommandOrRequest) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		p.Text = text
		return nil
	}

	var c Command
	if err := json.Unmarshal(data, &c); err == nil && c.Name != "" {
		p.Command = &c
		return nil
	}

	var r Request
	if err := json.Unmarshal(data, &r); err == nil && r.Url != "" {
		p.Request = &r
		return nil
	}

	return errors.New("page must be a string, a command or a request")
}

func (p TextOrCommandOrRequest) MarshalJSON() ([]byte, error) {
	if p.Text != "" {
		return json.Marshal(p.Text)
	}

	if p.Command != nil {
		return json.Marshal(p.Command)
	}

	if p.Request != nil {
		return json.Marshal(p.Request)
	}

	return nil, errors.New("page must be a string, a command or a request")
}

func NewReloadAction() Action {
	return Action{
		Type: ReloadAction,
	}
}

func NewCopyAction(title string, text string) Action {
	return Action{
		Title: title,
		Type:  CopyAction,
		Text:  text,
	}
}

func NewOpenAction(title string, target string) Action {
	return Action{
		Title:  title,
		Type:   OpenAction,
		Target: target,
	}
}

func NewPushAction(title string, page string) Action {
	return Action{
		Title: title,
		Type:  PushAction,
		Page: &PageGenerator{
			Path: page,
		},
	}
}

func NewRunAction(title string, name string, args ...string) Action {
	return Action{
		Title: title,
		Type:  RunAction,
		Command: &Command{
			Args: args,
		},
	}
}

type Command struct {
	Name  string   `json:"name"`
	Args  []string `json:"args,omitempty"`
	Input string   `json:"input,omitempty"`
	Dir   string   `json:"dir,omitempty"`
}

func (c Command) Cmd(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if c.Input != "" {
		cmd.Stdin = strings.NewReader(c.Input)
	}

	return cmd

}

func (c Command) Run(ctx context.Context) error {
	cmd := c.Cmd(ctx)

	var exitErr *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exitErr) {
		return fmt.Errorf("command exited with %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
	} else if err != nil {
		return err
	}

	return nil

}

func (c Command) Output(ctx context.Context) ([]byte, error) {
	cmd := c.Cmd(ctx)
	output, err := cmd.Output()

	var exitErr *exec.ExitError
	var pathErr *fs.PathError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("command exited with %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))

	} else if errors.As(err, &pathErr) {
		if strings.Contains(err.Error(), "permission denied") && runtime.GOOS != "windows" {
			return nil, fmt.Errorf("permission denied, try running `chmod +x %s`", c.Name)
		}
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("command failed: %T", err)
	}

	return output, nil
}

func (c *Command) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.Name = "bash"
		c.Args = []string{"-c", s}
		return nil
	}

	var args []string
	if err := json.Unmarshal(data, &args); err == nil {
		if len(args) == 0 {
			return fmt.Errorf("empty command")
		}
		c.Name = args[0]
		c.Args = args[1:]
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err == nil {
		if err := mapstructure.Decode(m, c); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("invalid command")
}
