package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/remote"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "测试远程服务器的代理连通性",
	Long:  `通过 SSH 连接远程服务器，测试代理隧道是否正常工作。`,
	RunE:  runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	s := remote.NewSetup(cfg, logger)
	if err := s.Connect(); err != nil {
		return err
	}
	defer s.Close()

	fmt.Println()
	fmt.Printf("测试代理: 远程 %s:%d\n", cfg.Server.Host, cfg.Tunnel.RemotePort)
	fmt.Println()

	results, err := s.RunTest()
	if err != nil {
		return fmt.Errorf("测试执行失败: %w", err)
	}

	fmt.Println("测试结果:")
	fmt.Println(results)
	return nil
}
