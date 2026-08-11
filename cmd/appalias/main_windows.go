//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DylonH78/AppAlias/internal/model"
	"github.com/DylonH78/AppAlias/internal/service"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var rootOverride string
	root := &cobra.Command{
		Use:           "appalias",
		Short:         "Create safe PowerShell start aliases for Windows applications",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&rootOverride, "root", "", "override the AppAlias data root (for portable use and tests)")
	newService := func(portable bool) (*service.Service, error) { return service.New(rootOverride, portable) }

	var portable bool
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "initialize AppAlias and add its bin directory to the user PATH",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(portable)
			if err != nil {
				return err
			}
			changed, err := svc.Init()
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintln(cmd.OutOrStdout(), "AppAlias was added to your user PATH. Open a new PowerShell window before using start <alias>.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "AppAlias is already present in your user PATH. Open a new PowerShell window if this session cannot find aliases.")
			}
			return nil
		},
	}
	initCmd.Flags().BoolVar(&portable, "portable", false, "use the directory containing appalias.exe as the application root")

	var scanJSON, scanApply bool
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "preview applications discovered from Windows launch sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			result := svc.Scan(context.Background())
			if scanApply {
				report, err := svc.Apply(result.Candidates)
				if err != nil {
					return err
				}
				return writeJSONOrText(cmd, scanJSON, report, func() {
					fmt.Fprintf(cmd.OutOrStdout(), "Applied: %s\n", strings.Join(report.Applied, ", "))
					for name, reason := range report.Skipped {
						fmt.Fprintf(cmd.OutOrStdout(), "Skipped %s: %s\n", name, reason)
					}
				})
			}
			return writeJSONOrText(cmd, scanJSON, result, func() {
				for _, candidate := range result.Candidates {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", candidate.Recommended, candidate.DisplayName, candidate.Source, candidate.Launch.Target)
				}
				for _, diagnostic := range result.Diagnostics {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", diagnostic)
				}
			})
		},
	}
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "write JSON output")
	scanCmd.Flags().BoolVar(&scanApply, "apply", false, "create safe, unique recommended aliases")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "list managed aliases",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			items, err := svc.List()
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", item.Name, item.DisplayName, item.Launch.Target)
			}
			return nil
		},
	}

	var addName, addTarget, addDisplay string
	var addArgs []string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "add a manually selected .exe application",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			return svc.Add(addName, addDisplay, model.LaunchSpec{Kind: model.LaunchExecutable, Target: addTarget, Arguments: addArgs})
		},
	}
	addCmd.Flags().StringVar(&addName, "name", "", "alias name")
	addCmd.Flags().StringVar(&addTarget, "target", "", "absolute path to a .exe file")
	addCmd.Flags().StringVar(&addDisplay, "display-name", "", "optional display name")
	addCmd.Flags().StringArrayVar(&addArgs, "arg", nil, "application argument; repeat for multiple arguments")
	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("target")

	renameCmd := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "rename a managed alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			return svc.Rename(args[0], args[1])
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <alias>",
		Short: "remove an alias and its launcher shim",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			return svc.Remove(args[0])
		},
	}

	launchCmd := &cobra.Command{
		Use:   "launch <alias>",
		Short: "launch a managed alias without using Start-Process",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			return svc.Launch(args[0])
		},
	}

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "check PATH, launchers, and application targets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			report, err := svc.Doctor()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Root: %s\nBin: %s\nPATH visible in this session: %t\nLauncher present: %t\n", report.Root, report.BinDirectory, report.PathInSession, report.LauncherPresent)
			for _, issue := range report.Issues {
				fmt.Fprintf(cmd.OutOrStdout(), "Issue: %s\n", issue)
			}
			return nil
		},
	}

	repairCmd := &cobra.Command{
		Use:   "repair",
		Short: "rebuild alias launcher shims after an update",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			items, err := svc.Repair()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Repaired: %s\n", strings.Join(items, ", "))
			return nil
		},
	}

	uninstallPathCmd := &cobra.Command{
		Use:    "uninstall-path",
		Short:  "remove AppAlias from the user PATH",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService(false)
			if err != nil {
				return err
			}
			changed, err := svc.RemoveFromUserPath()
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintln(cmd.OutOrStdout(), "AppAlias was removed from the user PATH.")
			}
			return nil
		},
	}

	guiCmd := &cobra.Command{
		Use:   "gui",
		Short: "open the AppAlias manager",
		RunE: func(_ *cobra.Command, _ []string) error {
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			guiExecutable := filepath.Join(filepath.Dir(executable), "appalias-gui.exe")
			if err := exec.Command(guiExecutable).Start(); err != nil {
				return fmt.Errorf("start AppAlias manager: %w", err)
			}
			return nil
		},
	}

	root.AddCommand(initCmd, scanCmd, listCmd, addCmd, renameCmd, removeCmd, launchCmd, doctorCmd, repairCmd, uninstallPathCmd, guiCmd)
	return root
}

func writeJSONOrText(cmd *cobra.Command, useJSON bool, value any, text func()) error {
	if !useJSON {
		text()
		return nil
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
