package main

import (
	"time"

	"github.com/spf13/cobra"
)

var calendarCmd = &cobra.Command{
	Use:     "calendar",
	Aliases: []string{"cal"},
	Short:   "Calendars and events",
}

func init() {
	rootCmd.AddCommand(calendarCmd)
	calendarCmd.AddCommand(calListCmd())
	calendarCmd.AddCommand(calEventsCmd())
	calendarCmd.AddCommand(calAddCmd())
	calendarCmd.AddCommand(calDeleteCmd())
}

func calListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			out, err := c.ListCalendars(cmd.Context(), "", "")
			if err != nil {
				return err
			}
			printTable([]string{"objectId", "title", "textColor", "backgroundColor", "isEditable"}, out)
			return nil
		},
	}
}

func calEventsCmd() *cobra.Command {
	var start, end string
	cmd := &cobra.Command{
		Use:   "events",
		Short: "List calendar events for a date range (default: next 7 days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			if start == "" || end == "" {
				start = time.Now().Format("2006-01-02")
				end = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
			}
			out, err := c.ListEvents(cmd.Context(), start, end)
			if err != nil {
				return err
			}
			printTable([]string{"objectId", "title", "start", "end", "allDayLong"}, out)
			return nil
		},
	}
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD")
	return cmd
}

func calAddCmd() *cobra.Command {
	var cal, desc string
	var allDay bool
	cmd := &cobra.Command{
		Use:   "add TITLE START END",
		Short: "Add a calendar event",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			ev, err := c.AddEvent(cmd.Context(), cal, args[0], args[1], args[2], desc, allDay)
			if err != nil {
				return err
			}
			printObject(ev)
			return nil
		},
	}
	cmd.Flags().StringVar(&cal, "calendar", "", "calendar id (default from env OO_CALENDAR_ID)")
	cmd.Flags().StringVar(&desc, "description", "", "event description")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "mark as all-day event")
	return cmd
}

func calDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete EVENT_ID [EVENT_ID...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more calendar events",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO(cmd)
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteEvent(cmd.Context(), id)
				if err != nil {
					return err
				}
				printObject(out)
			}
			return nil
		},
	}
}
