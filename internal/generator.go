package internal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/google/shlex"
	"github.com/pomdtr/sunbeam/schemas"
	"github.com/pomdtr/sunbeam/types"

	"gopkg.in/yaml.v3"
)

type PageGenerator func(input string) ([]byte, error)

type CmdGenerator struct {
	Command string
	Args    []string
	Dir     string
}

func NewActionGenerator(action types.Action, inputs map[string]string) PageGenerator {
	for key, value := range inputs {
		action.Command = strings.ReplaceAll(action.Command, fmt.Sprintf("${input:%s}", key), shellescape.Quote(value))
		action.Url = strings.ReplaceAll(action.Url, fmt.Sprintf("${input:%s}", key), url.QueryEscape(value))
		for i, header := range action.Headers {
			action.Headers[i] = strings.ReplaceAll(header, fmt.Sprintf("${input:%s}", key), value)
		}
		action.Text = strings.ReplaceAll(action.Text, fmt.Sprintf("${input:%s}", key), value)
		action.Path = strings.ReplaceAll(action.Path, fmt.Sprintf("${input:%s}", key), value)
	}

	for _, env := range os.Environ() {
		tokens := strings.SplitN(env, "=", 2)
		if len(tokens) != 2 {
			continue
		}
		action.Command = strings.ReplaceAll(action.Command, fmt.Sprintf("${env:%s}", tokens[0]), shellescape.Quote(tokens[1]))
		action.Url = strings.ReplaceAll(action.Url, fmt.Sprintf("${env:%s}", tokens[0]), url.QueryEscape(tokens[1]))
		for i, header := range action.Headers {
			action.Headers[i] = strings.ReplaceAll(header, fmt.Sprintf("${env:%s}", tokens[0]), tokens[1])
		}
		action.Text = strings.ReplaceAll(action.Text, fmt.Sprintf("${env:%s}", tokens[0]), tokens[1])
		action.Path = strings.ReplaceAll(action.Path, fmt.Sprintf("${env:%s}", tokens[0]), tokens[1])
	}

	switch action.Type {
	case types.RunAction:
		return NewCommandGenerator(action.Command, "", action.Dir)
	case types.FetchAction:
		return NewHttpGenerator(action.Url, action.Method, action.Headers, action.Body)
	case types.ReadAction:
		return NewFileGenerator(action.Path)
	}

	return nil
}

func NewCommandGenerator(command string, input string, dir string) PageGenerator {
	return func(query string) ([]byte, error) {
		command := strings.ReplaceAll(command, "${query}", shellescape.Quote(input))

		args, err := shlex.Split(command)
		if err != nil {
			return nil, err
		}

		if len(args) == 0 {
			return nil, fmt.Errorf("no command provided")
		}

		var extraArgs []string
		if len(args) > 1 {
			extraArgs = args[1:]
		}

		cmd := exec.Command(args[0], extraArgs...)
		cmd.Stdin = strings.NewReader(input)
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(input)
		output, err := cmd.Output()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("script exited with code %d: %s", exitError.ExitCode(), string(exitError.Stderr))
			}

			return nil, err
		}

		var v any
		if err := json.Unmarshal(output, &v); err != nil {
			return nil, err
		}

		if err := schemas.Validate(v); err != nil {
			return nil, err
		}

		var page types.Page
		if err := json.Unmarshal(output, &page); err != nil {
			return nil, err
		}

		if err := expandPage(&page, &url.URL{
			Scheme: "file",
			Path:   dir,
		}); err != nil {
			return nil, err
		}
		return output, nil
	}
}

func NewFileGenerator(name string) PageGenerator {
	return func(input string) ([]byte, error) {
		var page types.Page
		if path.Ext(name) == ".json" {
			bytes, err := os.ReadFile(name)
			if err != nil {
				return nil, err
			}

			var v any
			if err := json.Unmarshal(bytes, &v); err != nil {
				return nil, err
			}

			if err := schemas.Validate(v); err != nil {
				return nil, err
			}

			if err := json.Unmarshal(bytes, &page); err != nil {
				return nil, err
			}
		} else if path.Ext(name) == ".yaml" || path.Ext(name) == ".yml" {
			bytes, err := os.ReadFile(name)
			if err != nil {
				return nil, err
			}

			var v any
			if err := yaml.Unmarshal(bytes, &v); err != nil {
				return nil, err
			}

			if err := schemas.Validate(v); err != nil {
				return nil, err
			}

			if err := yaml.Unmarshal(bytes, &page); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupported file type")
		}

		if err := expandPage(&page, &url.URL{
			Scheme: "file",
			Path:   filepath.Dir(name),
		}); err != nil {
			return nil, err
		}

		return json.Marshal(page)
	}
}

func NewHttpGenerator(target string, method string, headers map[string]string, body string) PageGenerator {
	return func(query string) ([]byte, error) {
		target = strings.Replace(target, "${query}", url.QueryEscape(query), -1)
		body = strings.Replace(body, "${query}", query, -1)
		for key, value := range headers {
			headers[key] = strings.Replace(value, "${query}", value, -1)
		}

		req, err := http.NewRequest(method, target, strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("could not create request: %w", err)
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("could not make request: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read response body: %w", err)
		}

		var v any
		if err := json.Unmarshal(bytes, &v); err != nil {
			return nil, err
		}

		if err := schemas.Validate(v); err != nil {
			return nil, err
		}

		var page types.Page
		if err := json.Unmarshal(bytes, &page); err != nil {
			return nil, err
		}

		target, err := url.Parse(target)
		if err != nil {
			return nil, err
		}

		if err := expandPage(&page, target); err != nil {
			return nil, err
		}

		return json.Marshal(page)
	}
}

func expandPage(page *types.Page, root *url.URL) error {
	switch page.Type {
	case types.DetailPage:
		if page.Dir == "" {
			page.Dir = root.Path
		}
	}
	for _, item := range page.Items {
		if item.Detail != nil {
			item.Detail.Dir = root.Path
		}
		for j, action := range item.Actions {
			switch action.Type {
			case types.CopyAction:
				if action.Title == "" {
					item.Actions[j].Title = "Copy"
				}
			case types.FetchAction:
				if action.Method == "" {
					item.Actions[j].Method = "Fetch"
				}
				target, err := url.Parse(action.Url)
				if !filepath.IsAbs(target.Path) && err == nil {
					item.Actions[j].Url = (&url.URL{
						Scheme: root.Scheme,
						Host:   root.Host,
						Path:   filepath.Join(root.Path, target.Path),
					}).String()
				}
			case types.RunAction:
				if root.Scheme != "file" {
					return fmt.Errorf("run actions are only supported for file urls")
				}

				if action.Title == "" {
					item.Actions[j].Title = "Run"
				}
				if action.Dir == "" {
					item.Actions[j].Dir = root.Path
				}
			case types.ReadAction:
				if root.Scheme != "file" {
					return fmt.Errorf("read actions are only supported for file urls")
				}

				if action.Title == "" {
					item.Actions[j].Title = "Read"
				}

				if strings.HasPrefix(action.Path, "~/") {
					homeDir, _ := os.UserHomeDir()
					action.Path = filepath.Join(homeDir, action.Path[2:])
				} else if !filepath.IsAbs(action.Path) && action.Dir == "" {
					item.Actions[j].Path = filepath.Join(root.Path, action.Path)
				}
			case types.OpenAction:
				if action.Title == "" {
					item.Actions[j].Title = "Open"
				}

				if strings.HasPrefix(action.Path, "~/") {
					homeDir, _ := os.UserHomeDir()
					action.Path = filepath.Join(homeDir, action.Path[2:])
				} else if action.Path != "" && !filepath.IsAbs(action.Path) {
					item.Actions[j].Path = filepath.Join(root.Path, action.Path)
				}
			}
		}
	}

	return nil
}