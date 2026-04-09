package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "localtun",
	Short: "SSH 反向隧道与代理转发管理工具",
	Long: `LocalTUN — 通过 SSH 反向隧道将云服务器流量转发到本地代理。

支持自动配置远程服务器、启动/停止隧道、测试代理连通性。`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (默认 ~/.localtun/config.yaml)")
}
