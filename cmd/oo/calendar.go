package main

import (
	"time"

	"github.com/spf13/cobra"
)

func cmdCalList() *cobra.Command {
	return &cobra.Command{
		Use:   "cal-list",
		Short: "List calendars (default date span)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			out, err := c.ListCalendars(cmd.Context(), "", "")
			if err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

func cmdCalEvents() *cobra.Command {
	var start, end string
	cmd := &cobra.Command{
		Use:   "cal-events",
		Short: "List calendar data for period (default: next 7 days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
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
			printJSON(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&start, "start", "", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&end, "end", "", "end date YYYY-MM-DD")
	return cmd
}

func cmdCalAdd() *cobra.Command {
	var cal, desc string
	var allDay bool
	cmd := &cobra.Command{
		Use:   "cal-add TITLE START END",
		Short: "Add calendar event",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			ev, err := c.AddEvent(cmd.Context(), cal, args[0], args[1], args[2], desc, allDay)
			if err != nil {
				return err
			}
			printJSON(ev)
			return nil
		},
	}
	cmd.Flags().StringVar(&cal, "calendar", "", "calendar id (default from env)")
	cmd.Flags().StringVar(&desc, "description", "", "")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "")
	return cmd
}

func cmdCalDel() *cobra.Command {
	return &cobra.Command{
		Use:   "cal-delete EVENT_ID [EVENT_ID...]",
		Short: "Delete calendar events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newOO()
			if err != nil {
				return err
			}
			for _, id := range args {
				out, err := c.DeleteEvent(cmd.Context(), id)
				if err != nil {
					return err
				}
				printJSON(out)
			}
			return nil
		},
	}
}
