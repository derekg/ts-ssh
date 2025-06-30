package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	
	"github.com/derekg/ts-ssh/internal/crypto/pqc"
)

// Style definitions using lipgloss
var (
	// Theme colors
	primaryColor   = lipgloss.Color("#04B575")
	errorColor     = lipgloss.Color("#FF4B4B")
	warningColor   = lipgloss.Color("#FFA500")
	infoColor      = lipgloss.Color("#3B82F6")
	
	// Styles
	titleStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)
	
	successStyle = lipgloss.NewStyle().
		Foreground(primaryColor)
		
	errorStyle = lipgloss.NewStyle().
		Foreground(errorColor).
		Bold(true)
		
	warningStyle = lipgloss.NewStyle().
		Foreground(warningColor)
		
	infoStyle = lipgloss.NewStyle().
		Foreground(infoColor)
		
	headerStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Underline(true)
)

// NewRootCmd creates the root command with Cobra/Fang integration
func NewRootCmd() *cobra.Command {
	config := &Config{
		EnablePQC: true,
		PQCLevel:  1,
	}
	
	rootCmd := &cobra.Command{
		Use:   "ts-ssh [user@]hostname[:port] [command...]",
		Short: T("root_short"),
		Long:  titleStyle.Render("ts-ssh") + " - " + T("root_long"),
		Example: T("root_examples"),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: if args are provided and first arg is not a subcommand,
			// treat it as a connection attempt
			if len(args) > 0 {
				return runConnect(config, args)
			}
			// Otherwise show help
			return cmd.Help()
		},
	}
	
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&config.SSHUser, "user", "u", "", T("flag_user_help"))
	rootCmd.PersistentFlags().StringVarP(&config.SSHKeyPath, "identity", "i", "", T("flag_identity_help"))
	rootCmd.PersistentFlags().StringVarP(&config.SSHConfigFile, "config", "F", "", T("flag_config_help"))
	rootCmd.PersistentFlags().StringVar(&config.TsnetDir, "tsnet-dir", "", T("flag_tsnet_help"))
	rootCmd.PersistentFlags().StringVar(&config.TsControlURL, "control-url", "", T("flag_control_help"))
	rootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, T("flag_verbose_help"))
	rootCmd.PersistentFlags().BoolVar(&config.InsecureHostKey, "insecure", false, T("flag_insecure_help"))
	rootCmd.PersistentFlags().BoolVar(&config.ForceInsecure, "force-insecure", false, T("flag_force_insecure_help"))
	rootCmd.PersistentFlags().StringVar(&config.Language, "lang", "", T("flag_lang_help"))
	rootCmd.PersistentFlags().BoolVar(&config.EnablePQC, "pqc", true, T("flag_pqc_help"))
	rootCmd.PersistentFlags().IntVar(&config.PQCLevel, "pqc-level", 1, T("flag_pqc_level_help"))
	
	// Add subcommands
	rootCmd.AddCommand(
		newConnectCmd(config),
		newSCPCmd(config),
		newListCmd(config),
		newExecCmd(config),
		newMultiCmd(config),
		newConfigCmd(config),
		newPQCCmd(config),
		newVersionCmd(config),
	)
	
	return rootCmd
}

// newConnectCmd creates the connect subcommand
func newConnectCmd(config *Config) *cobra.Command {
	var forwardDest string
	
	cmd := &cobra.Command{
		Use:     "connect [user@]hostname[:port] [command...]",
		Aliases: []string{"ssh"},
		Short:   T("connect_short"),
		Long:    T("connect_long"),
		Example: T("connect_examples"),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create connect command with forward destination
			connectCmd := &ConnectCommand{
				Config:      config,
				ForwardDest: forwardDest,
			}
			if len(args) > 0 {
				connectCmd.Target = args[0]
				if len(args) > 1 {
					connectCmd.Command = args[1:]
				}
			}
			return connectCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().StringVarP(&forwardDest, "forward", "W", "", "Forward stdin/stdout to specified destination")
	
	return cmd
}

// newSCPCmd creates the SCP subcommand
func newSCPCmd(config *Config) *cobra.Command {
	var recursive bool
	var preserve bool
	
	cmd := &cobra.Command{
		Use:   "scp [-r] [-p] source destination",
		Short: T("scp_short"),
		Long:  T("scp_long"),
		Example: T("scp_examples"),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scpCmd := &SCPCommand{
				Config:      config,
				Source:      args[0],
				Destination: args[1],
				Recursive:   recursive,
				Preserve:    preserve,
			}
			return scpCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively copy directories")
	cmd.Flags().BoolVarP(&preserve, "preserve", "p", false, "Preserve file attributes")
	
	return cmd
}

// newListCmd creates the list subcommand
func newListCmd(config *Config) *cobra.Command {
	var interactive bool
	
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   T("list_short"),
		Long:    T("list_long"),
		Example: T("list_examples"),
		RunE: func(cmd *cobra.Command, args []string) error {
			listCmd := &ListCommand{
				Config:      config,
				Interactive: interactive,
			}
			return listCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive host picker with styled UI")
	
	return cmd
}

// newExecCmd creates the exec subcommand
func newExecCmd(config *Config) *cobra.Command {
	var command string
	var parallel bool
	
	cmd := &cobra.Command{
		Use:   "exec [hosts...] -c command",
		Short: T("exec_short"),
		Long:  T("exec_long"),
		Example: T("exec_examples"),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			execCmd := &ExecCommand{
				Config:   config,
				Command:  command,
				Hosts:    args,
				Parallel: parallel,
			}
			return execCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute on hosts (required)")
	cmd.MarkFlagRequired("command")
	cmd.Flags().BoolVarP(&parallel, "parallel", "p", false, "Execute commands in parallel")
	
	return cmd
}

// newMultiCmd creates the multi subcommand
func newMultiCmd(config *Config) *cobra.Command {
	var hosts string
	var sessions bool
	var tmux bool
	
	cmd := &cobra.Command{
		Use:   "multi",
		Short: T("multi_short"),
		Long:  T("multi_long"),
		Example: T("multi_examples"),
		RunE: func(cmd *cobra.Command, args []string) error {
			multiCmd := &MultiCommand{
				Config:   config,
				Hosts:    hosts,
				Sessions: sessions,
				Tmux:     tmux,
			}
			return multiCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().StringVar(&hosts, "hosts", "", "Comma-separated list of hosts")
	cmd.Flags().BoolVarP(&sessions, "sessions", "s", false, "Create multiple SSH sessions")
	cmd.Flags().BoolVarP(&tmux, "tmux", "t", false, "Use tmux for session management")
	
	return cmd
}

// newConfigCmd creates the config subcommand
func newConfigCmd(config *Config) *cobra.Command {
	var show bool
	var set string
	var unset string
	var reset bool
	var global bool
	
	cmd := &cobra.Command{
		Use:   "config",
		Short: T("config_short"),
		Long:  T("config_long"),
		Example: T("config_examples"),
		RunE: func(cmd *cobra.Command, args []string) error {
			configCmd := &ConfigCommand{
				Config: config,
				Show:   show,
				Set:    set,
				Unset:  unset,
				Reset:  reset,
				Global: global,
			}
			return configCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().BoolVar(&show, "show", false, "Show current configuration")
	cmd.Flags().StringVar(&set, "set", "", "Set configuration value (key=value)")
	cmd.Flags().StringVar(&unset, "unset", "", "Unset configuration value")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset configuration to defaults")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "Apply to global configuration")
	
	return cmd
}

// newPQCCmd creates the PQC subcommand
func newPQCCmd(config *Config) *cobra.Command {
	var report bool
	var test bool
	var benchmark bool
	var showSupported bool
	
	cmd := &cobra.Command{
		Use:   "pqc",
		Short: T("pqc_short"),
		Long:  T("pqc_long"),
		Example: T("pqc_examples"),
		RunE: func(cmd *cobra.Command, args []string) error {
			pqcCmd := &PQCCommand{
				Config:        config,
				Report:        report,
				Test:          test,
				Benchmark:     benchmark,
				ShowSupported: showSupported,
			}
			return pqcCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().BoolVarP(&report, "report", "r", false, "Generate PQC usage report")
	cmd.Flags().BoolVarP(&test, "test", "t", false, "Test PQC functionality")
	cmd.Flags().BoolVarP(&benchmark, "benchmark", "b", false, "Run PQC performance benchmarks")
	cmd.Flags().BoolVarP(&showSupported, "supported", "s", false, "Show supported PQC algorithms")
	
	return cmd
}

// newVersionCmd creates the version subcommand
func newVersionCmd(config *Config) *cobra.Command {
	var short bool
	var commit bool
	
	cmd := &cobra.Command{
		Use:   "version",
		Short: T("version_short"),
		Long:  T("version_long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			versionCmd := &VersionCommand{
				Config: config,
				Short:  short,
				Commit: commit,
			}
			return versionCmd.Run(context.Background())
		},
	}
	
	cmd.Flags().BoolVarP(&short, "short", "s", false, "Show short version only")
	cmd.Flags().BoolVarP(&commit, "commit", "c", false, "Include commit information")
	
	return cmd
}

// runConnect handles the main SSH connection logic for backwards compatibility
func runConnect(config *Config, args []string) error {
	// Parse target and command
	if len(args) == 0 {
		return fmt.Errorf("target hostname required")
	}
	
	target := args[0]
	var command []string
	if len(args) > 1 {
		command = args[1:]
	}
	
	// Create connect command
	connectCmd := &ConnectCommand{
		Config:  config,
		Target:  target,
		Command: command,
	}
	
	// Show styled connection message
	if config.Verbose {
		fmt.Println(infoStyle.Render("🔐 Establishing secure connection to " + target + "..."))
	}
	
	// Run the connection
	return connectCmd.Run(context.Background())
}

// EnhancedListCommand shows an interactive host picker using huh
func (c *ListCommand) RunInteractive(ctx context.Context, hosts []string) error {
	if len(hosts) == 0 {
		fmt.Println(warningStyle.Render("⚠️  No hosts found on the Tailscale network"))
		return nil
	}
	
	// Create styled options
	options := make([]huh.Option[string], len(hosts))
	for i, host := range hosts {
		// Add some visual flair to the host display
		displayName := fmt.Sprintf("🖥️  %s", host)
		options[i] = huh.NewOption(displayName, host)
	}
	
	var selectedHost string
	
	// Create the interactive form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(headerStyle.Render("Select a host to connect to")).
				Description("Use arrow keys to navigate, Enter to select").
				Options(options...).
				Value(&selectedHost),
		),
	)
	
	// Run the form
	if err := form.Run(); err != nil {
		return err
	}
	
	if selectedHost == "" {
		return fmt.Errorf("no host selected")
	}
	
	// Show connection message
	fmt.Println(successStyle.Render("✓ Selected: " + selectedHost))
	fmt.Println(infoStyle.Render("🔐 Establishing connection..."))
	
	// Create app config for SSH connection
	appConfig := &AppConfig{
		SSHUser:         c.SSHUser,
		SSHKeyPath:      c.SSHKeyPath,
		TsnetDir:        c.TsnetDir,
		TsControlURL:    c.TsControlURL,
		Target:          selectedHost,
		Verbose:         c.Verbose,
		InsecureHostKey: c.InsecureHostKey,
		EnablePQC:       c.EnablePQC,
		PQCLevel:        c.PQCLevel,
	}
	
	// Set up logger
	if c.Verbose {
		appConfig.Logger = logger
	} else {
		appConfig.Logger = log.New(io.Discard, "", 0)
	}
	
	return handleSSHOperation(appConfig)
}

// ShowPQCReport displays a styled PQC report
func (c *PQCCommand) ShowStyledReport(logger *log.Logger) error {
	report := pqc.GenerateGlobalReport(logger)
	
	// Style the report header
	fmt.Println(headerStyle.Render("🔐 Post-Quantum Cryptography Report"))
	fmt.Println()
	
	// Parse and style the report sections
	lines := strings.Split(report, "\n")
	for _, line := range lines {
		if strings.Contains(line, "✓") {
			fmt.Println(successStyle.Render(line))
		} else if strings.Contains(line, "⚠") {
			fmt.Println(warningStyle.Render(line))
		} else if strings.Contains(line, "❌") {
			fmt.Println(errorStyle.Render(line))
		} else if strings.HasPrefix(line, "=") || strings.HasPrefix(line, "-") {
			fmt.Println(infoStyle.Render(line))
		} else {
			fmt.Println(line)
		}
	}
	
	// Check quantum readiness
	ready, assessment := pqc.CheckGlobalQuantumReadiness(logger)
	fmt.Println()
	
	if ready {
		fmt.Println(successStyle.Render("✅ Quantum Readiness: " + assessment))
	} else {
		fmt.Println(warningStyle.Render("⚠️  Quantum Readiness: " + assessment))
	}
	
	// Get recommendations
	recommendations := pqc.GetGlobalRecommendations(logger)
	if len(recommendations) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("📋 Recommendations"))
		for _, rec := range recommendations {
			fmt.Printf("  %s %s\n", infoStyle.Render("•"), rec)
		}
	}
	
	return nil
}

// ExecuteWithFang runs the CLI with Fang enhancements
func ExecuteWithFang(ctx context.Context) error {
	// Initialize i18n early based on command line arguments
	initI18nForCLI(os.Args)
	
	rootCmd := NewRootCmd()
	
	// Apply Fang enhancements
	return fang.Execute(ctx, rootCmd)
}