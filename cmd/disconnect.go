package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"localtun/internal/console"
	sessionstore "localtun/internal/session"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect <session-id>",
	Short: "停止 LocalTUN Next 后台 session",
	Args:  cobra.ExactArgs(1),
	RunE:  runDisconnect,
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}

func runDisconnect(cmd *cobra.Command, args []string) error {
	ui := console.ForStdout()
	store, err := sessionstore.DefaultStore()
	if err != nil {
		return err
	}
	meta, err := store.Load(args[0])
	if err != nil {
		return err
	}
	if meta.PID > 0 {
		process, err := os.FindProcess(meta.PID)
		if err == nil {
			_ = process.Kill()
		}
	}
	if err := store.Remove(meta.ID); err != nil {
		return err
	}
	fmt.Printf("%s %s %s\n", ui.SuccessMark(), ui.Success("Disconnected LocalTUN session"), ui.Info(meta.ID))
	return nil
}
