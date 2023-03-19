package tui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/shlex"
	"github.com/pkg/browser"
	"github.com/pomdtr/sunbeam/types"
	"github.com/pomdtr/sunbeam/utils"
)

type CommandRunner struct {
	width, height int
	currentView   RunnerView

	Generator PageGenerator
	Validator PageValidator
	url       *url.URL

	header Header
	footer Footer

	form   *Form
	list   *List
	detail *Detail
	err    *Detail
}

type RunnerView int

const (
	RunnerViewList RunnerView = iota
	RunnerViewDetail
	RunnerViewLoading
)

type PageValidator func([]byte) error

func NewRunner(generator PageGenerator, validator PageValidator, url *url.URL) *CommandRunner {
	return &CommandRunner{
		header:      NewHeader(),
		footer:      NewFooter("Sunbeam"),
		currentView: RunnerViewLoading,
		Generator:   generator,
		Validator:   validator,
		url:         url,
	}

}
func (c *CommandRunner) Init() tea.Cmd {
	return tea.Batch(c.SetIsloading(true), c.Refresh)
}

type CommandOutput []byte

func (c *CommandRunner) Refresh() tea.Msg {
	var query string
	if c.currentView == RunnerViewList {
		query = c.list.Query()
	}

	output, err := c.Generator(query)
	if err != nil {
		return err
	}

	return CommandOutput(output)
}

func (runner *CommandRunner) handleAction(action types.Action) tea.Cmd {
	switch action.Type {
	case types.ReloadAction:
		return tea.Sequence(runner.SetIsloading(true), runner.Refresh)
	case types.EditAction:
		if runner.url.Scheme != "file" {
			return func() tea.Msg {
				return fmt.Errorf("cannot edit file on non-file url")
			}
		}
		if strings.HasPrefix(action.Path, "~") {
			home, _ := os.UserHomeDir()
			action.Path = path.Join(home, action.Path[1:])
		}
		editor, ok := os.LookupEnv("EDITOR")
		if !ok {
			editor = "vi"
		}

		return func() tea.Msg {
			return types.Action{
				Type:      types.RunAction,
				OnSuccess: types.ExitOnSuccess,
				Command:   fmt.Sprintf("%s %s", editor, shellescape.Quote(action.Path)),
			}
		}
	case types.OpenAction:
		var target string
		if action.Url != "" {
			target = action.Url
		} else if action.Path != "" {
			if runner.url.Scheme != "file" {
				return func() tea.Msg {
					return fmt.Errorf("cannot open file on non-file url")
				}
			}

			if strings.HasPrefix(action.Path, "~") {
				home, _ := os.UserHomeDir()
				target = path.Join(home, action.Path[1:])
			} else if !path.IsAbs(action.Path) {
				target = path.Join(runner.url.Path, action.Path)
			} else {
				target = action.Path
			}
		}

		if err := browser.OpenURL(target); err != nil {
			return func() tea.Msg {
				return err
			}
		}

		return tea.Quit
	case types.CopyAction:
		err := clipboard.WriteAll(action.Text)
		if err != nil {
			return func() tea.Msg {
				return fmt.Errorf("failed to copy text to clipboard: %s", err)
			}
		}

		return tea.Quit
	case types.ReadAction:
		var page string
		if runner.url.Scheme == "file" && path.IsAbs(action.Path) {
			page = action.Path
		} else if runner.url.Scheme == "file" && strings.HasPrefix(action.Path, "~") {
			home, _ := os.UserHomeDir()
			page = path.Join(home, action.Path[1:])
		} else {
			page = path.Join(runner.url.Path, action.Path)
		}

		runner := NewRunner(NewFileGenerator(
			page,
		), runner.Validator, &url.URL{
			Scheme: runner.url.Scheme,
			Path:   path.Dir(page),
		})
		return func() tea.Msg {
			return PushPageMsg{runner: runner}
		}
	case types.HttpAction:
		target, err := url.Parse(action.Url)
		if err != nil {
			return func() tea.Msg {
				return fmt.Errorf("failed to parse url: %s", err)
			}
		}
		if target.Scheme == "" && runner.url.Scheme != "" {
			target = &url.URL{
				Scheme: runner.url.Scheme,
				Host:   runner.url.Host,
				Path:   path.Join(runner.url.Path, target.Path),
			}
		}

		return func() tea.Msg {
			return PushPageMsg{
				runner: NewRunner(NewHttpGenerator(action.Url, action.Method, action.Headers, action.Body), runner.Validator, &url.URL{
					Scheme: target.Scheme,
					Host:   target.Host,
					Path:   path.Dir(target.Path),
				}),
			}
		}

	case types.RunAction:
		if runner.url.Scheme != "file" {
			return func() tea.Msg {
				return fmt.Errorf("cannot run command on non-file url")
			}
		}

		generator := NewCommandGenerator(action.Command, "", runner.url.Path)
		switch action.OnSuccess {
		case types.PushOnSuccess:
			runner := NewRunner(generator, runner.Validator, runner.url)
			return func() tea.Msg {
				return PushPageMsg{runner: runner}
			}
		case types.ReloadOnSuccess:
			return func() tea.Msg {
				_, err := generator("")
				if err != nil {
					if err, ok := err.(*exec.ExitError); ok {
						return fmt.Errorf("command exit with code %d: %s", err.ExitCode(), err.Stderr)
					}
					return err
				}

				return types.Action{
					Type: types.ReloadAction,
				}
			}
		case types.ReplaceOnSuccess:
			runner.Generator = generator
			return runner.Refresh

		case types.ExitOnSuccess:
			args, err := shlex.Split(action.Command)
			if err != nil {
				return func() tea.Msg {
					return fmt.Errorf("failed to parse command: %s", err)
				}
			}

			var extraArgs []string
			if len(args) > 1 {
				extraArgs = args[1:]
			}

			command := exec.Command(args[0], extraArgs...)
			command.Dir = runner.url.Path

			return func() tea.Msg {
				return command
			}
		default:
			return func() tea.Msg {
				return fmt.Errorf("unsupported onSuccess")
			}
		}
	default:
		return func() tea.Msg {
			return fmt.Errorf("unknown action type")
		}
	}
}

func (c *CommandRunner) SetIsloading(isLoading bool) tea.Cmd {
	switch c.currentView {
	case RunnerViewList:
		return c.list.SetIsLoading(isLoading)
	case RunnerViewDetail:
		return c.detail.SetIsLoading(isLoading)
	case RunnerViewLoading:
		return c.header.SetIsLoading(isLoading)
	default:
		return nil
	}
}

func (c *CommandRunner) SetSize(width, height int) {
	c.width, c.height = width, height

	c.header.Width = width
	c.footer.Width = width

	if c.form != nil {
		c.form.SetSize(width, height)
		return
	}

	if c.err != nil {
		c.err.SetSize(width, height)
		return
	}

	switch c.currentView {
	case RunnerViewList:
		c.list.SetSize(width, height)
	case RunnerViewDetail:
		c.detail.SetSize(width, height)
	}
}

func (runner *CommandRunner) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if runner.form != nil {
				runner.form = nil
				return runner, nil
			}

			if runner.currentView == RunnerViewLoading {
				return runner, func() tea.Msg {
					return PopPageMsg{}
				}
			}
		}
	case CommandOutput:
		runner.SetIsloading(false)
		if err := runner.Validator(msg); err != nil {
			return runner, func() tea.Msg {
				return fmt.Errorf("invalid response: %s", err)
			}
		}

		var page types.Page
		err := json.Unmarshal(msg, &page)
		if err != nil {
			return runner, func() tea.Msg {
				return err
			}
		}

		if page.Title == "" {
			page.Title = "Sunbeam"
		}

		switch page.Type {
		case types.DetailPage:
			var detailFunc func() string
			if page.Text != "" {
				detailFunc = func() string {
					return page.Text
				}
			} else if page.Command != "" {
				generator := NewCommandGenerator(page.Command, "", runner.url.Path)
				detailFunc = func() string {
					output, err := generator("")
					if err != nil {
						return err.Error()
					}
					return string(output)
				}
			} else {
				return runner, func() tea.Msg {
					return fmt.Errorf("detail page must have either text or command")
				}
			}

			runner.currentView = RunnerViewDetail
			runner.detail = NewDetail(page.Title, detailFunc, page.Actions)
			runner.detail.Language = page.Language
			runner.detail.SetSize(runner.width, runner.height)

			return runner, runner.detail.Init()
		case types.ListPage:
			runner.currentView = RunnerViewList

			// Save query string
			var query string
			var selectedId string

			if runner.list != nil {
				query = runner.list.Query()
				if runner.list.Selection() != nil {
					selectedId = runner.list.Selection().ID()
				}
			}

			runner.list = NewList(page, runner.url.Path)
			runner.list.SetQuery(query)

			listItems := make([]ListItem, len(page.Items))
			for i, scriptItem := range page.Items {
				scriptItem := scriptItem
				listItem := ParseScriptItem(scriptItem)
				listItems[i] = listItem
			}

			cmd := runner.list.SetItems(listItems, selectedId)

			runner.list.SetSize(runner.width, runner.height)
			return runner, tea.Sequence(runner.list.Init(), cmd)
		}

	case types.Action:
		if len(msg.Inputs) > 0 {
			formItems := make([]FormItem, len(msg.Inputs))
			for i, input := range msg.Inputs {
				item, err := NewFormItem(input)
				if err != nil {
					return runner, func() tea.Msg {
						return fmt.Errorf("failed to create form input: %s", err)
					}
				}

				formItems[i] = item
			}

			form := NewForm(formItems, func(values map[string]string) tea.Cmd {
				for key, value := range values {
					msg.Command = strings.ReplaceAll(msg.Command, fmt.Sprintf("${input:%s}", key), shellescape.Quote(value))
					msg.Url = strings.ReplaceAll(msg.Url, fmt.Sprintf("${input:%s}", key), url.QueryEscape(value))
					for i, header := range msg.Headers {
						msg.Headers[i] = strings.ReplaceAll(header, fmt.Sprintf("${input:%s}", key), value)
					}
					msg.Text = strings.ReplaceAll(msg.Text, fmt.Sprintf("${input:%s}", key), value)
					msg.Path = strings.ReplaceAll(msg.Path, fmt.Sprintf("${input:%s}", key), value)
				}

				msg.Inputs = nil
				return func() tea.Msg {
					return msg
				}
			})

			runner.form = form
			runner.SetSize(runner.width, runner.height)
			return runner, form.Init()
		}

		runner.form = nil
		cmd := runner.handleAction(msg)
		return runner, cmd
	case error:
		errorView := NewDetail("Error", msg.Error, []types.Action{
			{
				Type:  types.CopyAction,
				Title: "Copy error",
				Text:  msg.Error(),
			},
		})

		runner.err = errorView
		runner.err.SetSize(runner.width, runner.height)
		return runner, runner.err.Init()
	}

	var cmd tea.Cmd
	var container Page

	if runner.form != nil {
		container, cmd = runner.form.Update(msg)
		runner.form, _ = container.(*Form)
		return runner, cmd
	}

	if runner.err != nil {
		container, cmd = runner.err.Update(msg)
		runner.err, _ = container.(*Detail)
		return runner, cmd
	}

	switch runner.currentView {
	case RunnerViewList:
		container, cmd = runner.list.Update(msg)
		runner.list, _ = container.(*List)
	case RunnerViewDetail:
		container, cmd = runner.detail.Update(msg)
		runner.detail, _ = container.(*Detail)
	default:
		runner.header, cmd = runner.header.Update(msg)
	}
	return runner, cmd
}

func (c *CommandRunner) View() string {
	if c.form != nil {
		return c.form.View()
	}

	if c.err != nil {
		return c.err.View()
	}

	switch c.currentView {
	case RunnerViewList:
		return c.list.View()
	case RunnerViewDetail:
		return c.detail.View()
	case RunnerViewLoading:
		headerView := c.header.View()
		footerView := c.footer.View()
		padding := make([]string, utils.Max(0, c.height-lipgloss.Height(headerView)-lipgloss.Height(footerView)))
		return lipgloss.JoinVertical(lipgloss.Left, c.header.View(), strings.Join(padding, "\n"), c.footer.View())
	default:
		return ""
	}
}
