package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"localtun/internal/console"
	sessionstore "localtun/internal/session"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "列出 LocalTUN Next 后台 sessions",
	RunE:  runSessions,
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
}

func runSessions(cmd *cobra.Command, args []string) error {
	ui := console.ForStdout()
	store, err := sessionstore.DefaultStore()
	if err != nil {
		return err
	}
	sessions, err := store.List()
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println(ui.Muted("No detached LocalTUN sessions."))
		return nil
	}
	for _, meta := range sessions {
		fmt.Printf("%s %s\n", ui.SuccessMark(), ui.Info(meta.ID))
		fmt.Printf("  %s %s\n", ui.Label("target:"), ui.Info(meta.Target))
		fmt.Printf("  %s %s\n", ui.Label("pid:"), ui.Accent(fmt.Sprint(meta.PID)))
		fmt.Printf("  %s %s\n", ui.Label("local proxy:"), ui.Accent(meta.LocalProxy))
		fmt.Printf("  %s %s\n", ui.Label("remote proxy:"), ui.Accent(meta.ProxyURL))
		fmt.Printf("  %s %s\n", ui.Label("created:"), ui.Muted(meta.CreatedAt.Local().Format("2006-01-02 15:04:05")))
	}
	return nil
}
