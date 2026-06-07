package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"localtun/internal/console"
)

var rootCmd = &cobra.Command{
	Use:   "localtun",
	Short: "Internet-enabled SSH sessions",
	Long: `LocalTUN Next — enter an SSH session that already has Internet access.

LocalTUN creates a temporary SSH reverse tunnel to your local proxy and injects
proxy environment variables only into the current remote shell session.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, console.ForStderr().Error("错误:"), err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
