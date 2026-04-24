package main

import (
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Portal / workspace users",
}

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersListCmd())
	usersCmd.AddCommand(usersSelfCmd())
	rootCmd.AddCommand(whoamiCmd())
}

func usersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List portal users",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			users, err := c.GetUsers()
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(users))
			for _, u := range users {
				rows = append(rows, map[string]any{
					"id":          derefString(u.ID),
					"userName":    derefString(u.UserName),
					"displayName": derefString(u.DisplayName),
					"email":       derefString(u.Email),
					"isAdmin":     derefBool(u.IsAdmin),
					"status":      derefInt(u.Status),
				})
			}
			printTable([]string{"id", "userName", "displayName", "email", "isAdmin", "status"}, rows)
			return nil
		},
	}
}

func usersSelfCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self",
		Short: "Show current user id (people/@self)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			id, err := c.SelfUserID(cmd.Context())
			if err != nil {
				return err
			}
			printObject(map[string]any{"id": id})
			return nil
		},
	}
}

// whoamiCmd is a convenience shortcut at the root level.
func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Alias for `oo users self`",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			id, err := c.SelfUserID(cmd.Context())
			if err != nil {
				return err
			}
			printObject(map[string]any{"id": id})
			return nil
		},
	}
}

func derefBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
