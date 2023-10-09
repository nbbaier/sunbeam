package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pomdtr/sunbeam/internal/tui"
	"github.com/pomdtr/sunbeam/internal/utils"
	"github.com/pomdtr/sunbeam/pkg/types"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func NewCmdExtension() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "extension",
		Aliases: []string{"ext", "extensions"},
		Short:   "Manage extensions",
		GroupID: CommandGroupCore,
	}

	cmd.AddCommand(NewCmdExtensionList())
	cmd.AddCommand(NewCmdExtensionInstall())
	cmd.AddCommand(NewCmdExtensionUpgrade())
	cmd.AddCommand(NewCmdExtensionRemove())
	cmd.AddCommand(NewCmdExtensionRename())
	cmd.AddCommand(NewCmdExtensionBrowse())

	return cmd
}

func NewCmdExtensionBrowse() *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Browse extensions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return utils.Open("https://github.com/topics/sunbeam-extension")
		},
	}
}

func NewCmdExtensionList() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List installed extensions",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			extensions, err := FindExtensions()
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				extensionMap := make(map[string]types.Manifest)
				for alias, extension := range extensions {
					extensionMap[alias] = extension.Manifest
				}

				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(extensionMap)
			}

			for alias, extension := range extensions {
				fmt.Printf("%s\t%s\n", alias, extension.Title)
			}

			return nil
		},
	}
}

func NewCmdExtensionInstall() *cobra.Command {
	flags := struct {
		alias string
	}{}

	cmd := &cobra.Command{
		Use:     "install <src>",
		Aliases: []string{"add"},
		Short:   "Install an extension",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			extensionRoot := filepath.Join(dataHome(), "extensions")
			if info, err := os.Stat(args[0]); err == nil {
				if !info.IsDir() {
					return fmt.Errorf("src must be a directory")
				}

				origin, err := filepath.Abs(args[0])
				if err != nil {
					return err
				}

				var alias string
				if flags.alias != "" {
					alias = flags.alias
				} else {
					alias = filepath.Base(origin)
					alias = strings.TrimSuffix(alias, filepath.Ext(alias))
					alias = strings.TrimPrefix(alias, "sunbeam-")
				}

				return installFromLocalDir(origin, filepath.Join(extensionRoot, alias))
			}

			origin := args[0]
			var alias string
			if flags.alias != "" {
				alias = flags.alias
			} else {
				parts := strings.Split(origin, "/")
				alias = parts[len(parts)-1]
				alias = strings.TrimSuffix(alias, filepath.Ext(alias))
				alias = strings.TrimPrefix(alias, "sunbeam-")
			}

			return installFromRepository(origin, filepath.Join(extensionRoot, alias))
		},
	}

	cmd.Flags().StringVarP(&flags.alias, "alias", "a", "", "alias for extension")
	return cmd
}

func installFromLocalDir(srcDir string, targetDir string) (err error) {
	originDir, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	entrypoint := filepath.Join(originDir, "sunbeam-extension")
	if _, err := os.Stat(entrypoint); err != nil {
		return fmt.Errorf("extension %s not found", srcDir)
	}

	extension, err := tui.LoadExtension(entrypoint)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(targetDir)
		}
	}()

	f, err := os.Create(filepath.Join(targetDir, "manifest.json"))
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(extension.Manifest); err != nil {
		return err
	}

	if err := os.Symlink(srcDir, filepath.Join(targetDir, "src")); err != nil {
		return err
	}

	return nil
}

func installFromRepository(origin string, targetDir string) (err error) {
	// check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found")
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(targetDir)
		}
	}()

	srcDir := filepath.Join(targetDir, "src")
	var cloneCmd *exec.Cmd
	if tag, err := getLatestTag(origin); err == nil {
		cloneCmd = exec.Command("git", "clone", "--depth=1", fmt.Sprintf("--branch=%s", tag), tag, origin, srcDir)
	} else {
		cloneCmd = exec.Command("git", "clone", "--depth=1", origin, srcDir)
	}

	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return err
	}

	extension, err := tui.LoadExtension(filepath.Join(srcDir, "sunbeam-extension"))
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(targetDir, "manifest.json"))
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(extension.Manifest); err != nil {
		return err
	}

	return nil
}

func getLatestTag(origin string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", origin)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	var tags []string
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		tags = append(tags, strings.TrimPrefix(parts[1], "refs/tags/"))
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}

	sort.SliceStable(tags, func(i, j int) bool {
		return semver.Compare(tags[i], tags[j]) == -1
	})

	return tags[len(tags)-1], nil
}

func NewCmdExtensionUpgrade() *cobra.Command {
	flags := struct {
		all bool
	}{}
	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Upgrade an extension",
		Args:    cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if flags.all {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			extensions, err := FindExtensions()
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}

			var completions []string
			for alias := range extensions {
				completions = append(completions, alias)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if flags.all && len(args) > 0 {
				return fmt.Errorf("cannot use --all with an extension")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// check if git is installed
			toUpgrade := make([]string, 0)
			if flags.all {
				extensions, err := FindExtensions()
				if err != nil {
					return err
				}

				for alias := range extensions {
					toUpgrade = append(toUpgrade, alias)
				}
			} else {
				toUpgrade = append(toUpgrade, args[0])
			}

			for _, alias := range toUpgrade {
				cmd.PrintErrln()
				cmd.PrintErrf("Upgrading %s...\n", alias)
				extensionDir := filepath.Join(dataHome(), "extensions", alias)

				if err := upgradeExtension(extensionDir); err != nil {
					cmd.PrintErrln(err)
				} else {
					cmd.PrintErrln("done")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flags.all, "all", "a", false, "upgrade all extensions")

	return cmd
}

func upgradeExtension(extensionDir string) error {
	f, err := os.Open(filepath.Join(extensionDir, "manifest.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	var manifest types.Manifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return err
	}
	f.Close()

	// check if extensionDir is a symlink
	if info, err := os.Lstat(extensionDir); err == nil && info.Mode()&os.ModeSymlink == 0 {
		getOriginCmd := exec.Command("git", "remote", "get-url", "origin")
		getOriginCmd.Dir = filepath.Join(extensionDir, "src")
		originBytes, err := getOriginCmd.Output()
		if err != nil {
			return err
		}

		origin := string(bytes.TrimSpace(originBytes))
		if tag, err := getLatestTag(origin); err == nil {
			checkoutCmd := exec.Command("git", "checkout", tag)
			checkoutCmd.Dir = filepath.Join(extensionDir, "src")
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			if err := checkoutCmd.Run(); err != nil {
				return err
			}

		} else {
			pullCmd := exec.Command("git", "pull")
			pullCmd.Dir = filepath.Join(extensionDir, "src")
			pullCmd.Stdout = os.Stdout
			pullCmd.Stderr = os.Stderr

			if err := pullCmd.Run(); err != nil {
				return err
			}
		}
	}

	// refresh manifest
	extension, err := tui.LoadExtension(filepath.Join(extensionDir, "src", "sunbeam-extension"))
	if err != nil {
		return err
	}

	f, err = os.Create(filepath.Join(extensionDir, "manifest.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(extension.Manifest); err != nil {
		return err
	}

	return nil
}

func NewCmdExtensionRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove an extension",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			extensions, err := FindExtensions()
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}

			var completions []string
			for alias := range extensions {
				completions = append(completions, alias)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			extensionDir := filepath.Join(dataHome(), "extensions", args[0])
			return os.RemoveAll(extensionDir)
		},
	}

	return cmd
}

func NewCmdExtensionRename() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename an extension",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			extensions, err := FindExtensions()
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}

			var completions []string
			for alias := range extensions {
				completions = append(completions, alias)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			extensionDir := filepath.Join(dataHome(), "extensions", args[0])
			newExtensionDir := filepath.Join(dataHome(), "extensions", args[1])

			return os.Rename(extensionDir, newExtensionDir)
		},
	}

	return cmd
}

func FindExtensions() (map[string]tui.Extension, error) {
	extensionRoot := filepath.Join(dataHome(), "extensions")
	if _, err := os.Stat(extensionRoot); err != nil {
		return nil, nil
	}

	entries, err := os.ReadDir(extensionRoot)
	if err != nil {
		return nil, err
	}
	extensionMap := make(map[string]tui.Extension)
	for _, entry := range entries {
		manifestPath := filepath.Join(extensionRoot, entry.Name(), "manifest.json")
		f, err := os.Open(manifestPath)
		if err != nil {
			continue
		}
		defer f.Close()

		var manifest types.Manifest
		if err := json.NewDecoder(f).Decode(&manifest); err != nil {
			continue
		}

		extensionMap[entry.Name()] = tui.Extension{
			Manifest:   manifest,
			Entrypoint: filepath.Join(extensionRoot, entry.Name(), "src", "sunbeam-extension"),
		}
	}

	return extensionMap, nil
}
