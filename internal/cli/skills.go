package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewSkillsCmd returns the "skills" command group.
func NewSkillsCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage and install Symbiont AI skills for Claude Code",
	}
	cmd.AddCommand(newSkillsListCmd(client))
	cmd.AddCommand(newSkillsInstallCmd(client))
	cmd.AddCommand(newSkillsUninstallCmd())
	return cmd
}

// newSkillsListCmd lists all available skills and their enabled state.
//
//	symbiont skills list
func newSkillsListCmd(client *APIClient) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills and their enabled state",
		RunE: func(cmd *cobra.Command, args []string) error {
			var resp struct {
				Skills []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Enabled     bool   `json:"enabled"`
				} `json:"skills"`
			}
			if err := client.Get(cmd.Context(), "/api/agent/skills", &resp); err != nil {
				return err
			}

			if IsJSON(cmd) {
				PrintJSON(resp)
				return nil
			}

			if len(resp.Skills) == 0 {
				fmt.Println("No skills available.")
				return nil
			}

			headers := []string{"NAME", "ENABLED", "DESCRIPTION"}
			rows := make([][]string, 0, len(resp.Skills))
			for _, s := range resp.Skills {
				enabled := "yes"
				if !s.Enabled {
					enabled = "no"
				}
				rows = append(rows, []string{s.Name, enabled, s.Description})
			}
			PrintTable(headers, rows)
			return nil
		},
	}
}

// defaultSkillsDir returns the default installation directory for skills.
func defaultSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills", "symbiont"), nil
}

// newSkillsInstallCmd writes enabled skills to the target directory.
//
//	symbiont skills install [--dir ~/.claude/skills/symbiont]
func newSkillsInstallCmd(client *APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install enabled skills to the Claude Code skills directory",
		Long: `Write enabled Symbiont skills to ~/.claude/skills/symbiont/ (or --dir).

Claude Code automatically discovers skills in that directory. Each enabled
skill is written as <name>.md with YAML frontmatter + workflow instructions.
Running install again overwrites existing files (idempotent).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				d, err := defaultSkillsDir()
				if err != nil {
					return err
				}
				dir = d
			}

			// Fetch list to know which skills are enabled.
			var listResp struct {
				Skills []struct {
					Name    string `json:"name"`
					Enabled bool   `json:"enabled"`
				} `json:"skills"`
			}
			if err := client.Get(cmd.Context(), "/api/agent/skills", &listResp); err != nil {
				return err
			}

			// Collect enabled skill names.
			var toInstall []string
			for _, s := range listResp.Skills {
				if s.Enabled {
					toInstall = append(toInstall, s.Name)
				}
			}
			// If none are explicitly enabled, install all (opt-out model).
			if len(toInstall) == 0 {
				for _, s := range listResp.Skills {
					toInstall = append(toInstall, s.Name)
				}
			}

			if len(toInstall) == 0 {
				fmt.Println("No skills to install.")
				return nil
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating skills directory %s: %w", dir, err)
			}

			installed := 0
			for _, name := range toInstall {
				var bodyResp struct {
					Body string `json:"body"`
				}
				if err := client.Get(cmd.Context(), "/api/agent/skills/"+name+"/body", &bodyResp); err != nil {
					fmt.Fprintf(os.Stderr, "  warning: could not fetch skill %q: %v\n", name, err)
					continue
				}

				dest := filepath.Join(dir, name+".md")
				if err := os.WriteFile(dest, []byte(bodyResp.Body), 0o644); err != nil {
					return fmt.Errorf("writing skill %q to %s: %w", name, dest, err)
				}
				fmt.Printf("  installed: %s\n", dest)
				installed++
			}

			fmt.Printf("\n%d skill(s) installed to %s\n", installed, dir)
			fmt.Printf("\nTo use in Claude Code, add to your skill search path or symlink:\n")
			fmt.Printf("  The skills directory is auto-discovered by Claude Code.\n")
			return nil
		},
	}
	cmd.Flags().String("dir", "", "installation directory (default: ~/.claude/skills/symbiont)")
	return cmd
}

// newSkillsUninstallCmd removes the skills directory.
//
//	symbiont skills uninstall [--dir ~/.claude/skills/symbiont]
func newSkillsUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the installed Symbiont skills directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				d, err := defaultSkillsDir()
				if err != nil {
					return err
				}
				dir = d
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				fmt.Printf("Remove skills directory %s? [y/N] ", dir)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("removing skills directory: %w", err)
			}
			fmt.Printf("Removed %s\n", dir)
			return nil
		},
	}
	cmd.Flags().String("dir", "", "skills directory to remove (default: ~/.claude/skills/symbiont)")
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	return cmd
}
