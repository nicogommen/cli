package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/platformsh/cli/internal"
	"github.com/platformsh/cli/internal/config"
	"github.com/platformsh/cli/internal/legacy"
)

// Execute is the main entrypoint to run the CLI.
func Execute(cnf *config.Config) error {
	ctx := config.ToContext(context.Background(), cnf)
	return newRootCommand(cnf).ExecuteContext(ctx)
}

func newRootCommand(cnf *config.Config) *cobra.Command {
	var (
		updateMessageChan = make(chan *internal.ReleaseInfo, 1)
		versionCommand    = newVersionCommand(cnf)
	)
	cmd := &cobra.Command{
		Use:                cnf.Application.Executable,
		Short:              cnf.Application.Name,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: false,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if viper.GetBool("version") {
				versionCommand.Run(cmd, []string{})
				os.Exit(0)
			}
			if cnf.Wrapper.GitHubRepo != "" {
				go func() {
					rel, _ := internal.CheckForUpdate(cnf.Wrapper.GitHubRepo, version)
					updateMessageChan <- rel
				}()
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			c := &legacy.CLIWrapper{
				Config:         cnf,
				Version:        version,
				CustomPharPath: viper.GetString("phar-path"),
				Debug:          viper.GetBool("debug"),
				Stdout:         cmd.OutOrStdout(),
				Stderr:         cmd.ErrOrStderr(),
				Stdin:          cmd.InOrStdin(),
			}
			if err := c.Init(); err != nil {
				debugLog("%s\n", color.RedString(err.Error()))
				os.Exit(1)
				return
			}

			if err := c.Exec(cmd.Context(), os.Args[1:]...); err != nil {
				debugLog("%s\n", color.RedString(err.Error()))
				exitCode := 1
				var execErr *exec.ExitError
				if errors.As(err, &execErr) {
					exitCode = execErr.ExitCode()
				}
				os.Exit(exitCode)
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			checkShellConfigLeftovers(cnf)
			select {
			case rel := <-updateMessageChan:
				printUpdateMessage(rel, cnf)
			default:
			}
		},
	}

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Context() == nil {
			cmd.SetContext(context.Background())
		}

		if !slices.Contains(args, "--help") && !slices.Contains(args, "-h") {
			args = append([]string{"help"}, args...)
		}

		cmd.Run(cmd, args)
	})

	cmd.PersistentFlags().BoolP("version", "V", false, fmt.Sprintf("Displays the %s version", cnf.Application.Name))
	cmd.PersistentFlags().String("phar-path", "",
		fmt.Sprintf("Uses a local .phar file for the Legacy %s", cnf.Application.Name),
	)
	cmd.PersistentFlags().Bool("debug", false, "Enable debug logging")

	// Add subcommands.
	cmd.AddCommand(
		newCompletionCommand(cnf),
		newHelpCommand(cnf),
		newListCommand(cnf),
		newProjectInitCommand(cnf),
		versionCommand,
	)

	//nolint:errcheck
	viper.BindPFlags(cmd.PersistentFlags())

	return cmd
}

// checkShellConfigLeftovers checks .zshrc and .bashrc for any leftovers from the legacy CLI
func checkShellConfigLeftovers(cnf *config.Config) {
	start := fmt.Sprintf("# BEGIN SNIPPET: %s configuration", cnf.Application.Name)
	end := "# END SNIPPET"
	shellConfigSnippet := regexp.MustCompile(regexp.QuoteMeta(start) + "(?s).+?" + regexp.QuoteMeta(end))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	shellConfigFiles := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bashrc"),
	}

	for _, shellConfigFile := range shellConfigFiles {
		if _, err := os.Stat(shellConfigFile); err != nil {
			continue
		}

		shellConfig, err := os.ReadFile(shellConfigFile)
		if err != nil {
			continue
		}

		if shellConfigSnippet.Match(shellConfig) {
			fmt.Fprintf(color.Error, "%s Your %s file contains code that is no longer needed for the New %s\n",
				color.YellowString("Warning:"),
				shellConfigFile,
				cnf.Application.Name,
			)
			fmt.Fprintf(color.Error, "%s %s\n", color.YellowString("Please remove the following lines from:"), shellConfigFile)
			fmt.Fprintf(color.Error, "\t%s\n", strings.ReplaceAll(string(shellConfigSnippet.Find(shellConfig)), "\n", "\n\t"))
		}
	}
}

func printUpdateMessage(newRelease *internal.ReleaseInfo, cnf *config.Config) {
	if newRelease == nil {
		return
	}

	fmt.Fprintf(color.Error, "\n\n%s %s → %s\n",
		color.YellowString(fmt.Sprintf("A new release of the %s is available:", cnf.Application.Name)),
		color.CyanString(version),
		color.CyanString(newRelease.Version),
	)

	if cnf.Wrapper.HomebrewTap != "" {
		executable, err := os.Executable()
		if err == nil && isUnderHomebrew(executable) {
			fmt.Fprintf(
				color.Error,
				"To upgrade, run: brew update && brew upgrade %s\n",
				color.YellowString(cnf.Wrapper.HomebrewTap),
			)
		}
	}

	fmt.Fprintf(color.Error, "%s\n\n", color.YellowString(newRelease.URL))
}

func isUnderHomebrew(binary string) bool {
	brewExe, err := exec.LookPath("brew")
	if err != nil {
		return false
	}

	brewPrefixBytes, err := exec.Command(brewExe, "--prefix").Output()
	if err != nil {
		return false
	}

	brewBinPrefix := filepath.Join(strings.TrimSpace(string(brewPrefixBytes)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(binary, brewBinPrefix)
}

func debugLog(format string, v ...any) {
	if !viper.GetBool("debug") {
		return
	}

	log.Printf(format, v...)
}
