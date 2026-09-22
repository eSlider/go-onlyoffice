package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	onlyoffice "github.com/eslider/go-onlyoffice"
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
	usersCmd.AddCommand(usersGetCmd())
	usersCmd.AddCommand(usersCreateCmd())
	usersCmd.AddCommand(usersUpdateCmd())
	usersCmd.AddCommand(usersDeleteCmd())
	usersCmd.AddCommand(usersBlockCmd())
	usersCmd.AddCommand(usersUnblockCmd())
	usersCmd.AddCommand(usersPasswordCmd())
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

func usersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get USER_ID",
		Short: "Show one portal user profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			u, err := c.GetUser(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printObject(u)
			return nil
		},
	}
}

func usersCreateCmd() *cobra.Command {
	var first, last, email, password, title, location, sex, comment string
	var visitor bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a portal user",
		Long: `Create a portal user (POST /api/2.0/people).

Without --password the portal generates one and the account stays NotActivated
until the user follows the activation link. With --password the account is
Active immediately. Use --visitor for a guest account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || first == "" || last == "" {
				return fmt.Errorf("--email, --first and --last are required")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			req := onlyoffice.NewUserRequest{
				FirstName: first,
				LastName:  last,
				Email:     email,
				Password:  password,
				Title:     title,
				Location:  location,
				Sex:       sex,
				Comment:   comment,
			}
			if cmd.Flags().Changed("visitor") {
				req.IsVisitor = &visitor
			}
			u, err := c.CreateUser(cmd.Context(), req)
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"id":          idString(u, "id"),
				"displayName": idString(u, "displayName"),
				"email":       idString(u, "email"),
				"status":      u["status"],
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&first, "first", "", "first name (required)")
	cmd.Flags().StringVar(&last, "last", "", "last name (required)")
	cmd.Flags().StringVar(&email, "email", "", "email (required)")
	cmd.Flags().StringVar(&password, "password", "", "initial password (default: portal-generated)")
	cmd.Flags().StringVar(&title, "title", "", "job title")
	cmd.Flags().StringVar(&location, "location", "", "location")
	cmd.Flags().StringVar(&sex, "sex", "", "sex: male|female")
	cmd.Flags().StringVar(&comment, "comment", "", "comment")
	cmd.Flags().BoolVar(&visitor, "visitor", false, "create as guest (isVisitor=true)")
	return cmd
}

func usersUpdateCmd() *cobra.Command {
	var first, last, email, title, location, sex, comment string
	cmd := &cobra.Command{
		Use:   "update USER_ID",
		Short: "Update portal user profile fields (only flags passed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("first") {
				body["firstname"] = first
			}
			if cmd.Flags().Changed("last") {
				body["lastname"] = last
			}
			if cmd.Flags().Changed("email") {
				body["email"] = email
			}
			if cmd.Flags().Changed("title") {
				body["title"] = title
			}
			if cmd.Flags().Changed("location") {
				body["location"] = location
			}
			if cmd.Flags().Changed("sex") {
				body["sex"] = sex
			}
			if cmd.Flags().Changed("comment") {
				body["comment"] = comment
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update: pass at least one of --first/--last/--email/--title/--location/--sex/--comment")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			u, err := c.UpdateUser(cmd.Context(), args[0], body)
			if err != nil {
				return err
			}
			printObject(map[string]any{
				"id":          idString(u, "id"),
				"displayName": idString(u, "displayName"),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&first, "first", "", "first name")
	cmd.Flags().StringVar(&last, "last", "", "last name")
	cmd.Flags().StringVar(&email, "email", "", "email")
	cmd.Flags().StringVar(&title, "title", "", "job title")
	cmd.Flags().StringVar(&location, "location", "", "location")
	cmd.Flags().StringVar(&sex, "sex", "", "sex: male|female")
	cmd.Flags().StringVar(&comment, "comment", "", "comment")
	return cmd
}

func usersDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete USER_ID [USER_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete portal user(s) permanently",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				u, err := c.DeleteUser(cmd.Context(), id)
				if err != nil {
					return fmt.Errorf("delete %s: %w", id, err)
				}
				printObject(map[string]any{"id": id, "deleted": true, "displayName": idString(u, "displayName")})
			}
			return nil
		},
	}
}

func usersBlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "block USER_ID [USER_ID...]",
		Aliases: []string{"disable"},
		Short:   "Block (terminate) user(s): login denied, profile kept",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				if err := c.BlockUser(cmd.Context(), id); err != nil {
					return fmt.Errorf("block %s: %w", id, err)
				}
				printObject(map[string]any{"id": id, "blocked": true})
			}
			return nil
		},
	}
}

func usersUnblockCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "unblock USER_ID [USER_ID...]",
		Aliases: []string{"enable", "activate"},
		Short:   "Unblock (activate) user(s)",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				if err := c.UnblockUser(cmd.Context(), id); err != nil {
					return fmt.Errorf("unblock %s: %w", id, err)
				}
				printObject(map[string]any{"id": id, "unblocked": true})
			}
			return nil
		},
	}
}

func usersPasswordCmd() *cobra.Command {
	var password string
	cmd := &cobra.Command{
		Use:   "password USER_ID",
		Short: "Set a user password (reads stdin when --password is empty)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd := password
			if pwd == "" {
				b, err := readLine(os.Stdin)
				if err != nil {
					return fmt.Errorf("read password: %w", err)
				}
				pwd = b
			}
			if pwd == "" {
				return fmt.Errorf("password is empty")
			}
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if err := c.ChangeUserPassword(cmd.Context(), args[0], pwd); err != nil {
				return err
			}
			printObject(map[string]any{"id": args[0], "password_changed": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "new password (omit to read one line from stdin)")
	return cmd
}

// readLine reads a single trimmed line from r.
func readLine(r *os.File) (string, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return strings.TrimSpace(sc.Text()), nil
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
