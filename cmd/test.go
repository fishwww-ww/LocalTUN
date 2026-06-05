package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
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

	ui := console.ForStdout()
	logger := log.New(os.Stdout, ui.Prefix("test"), log.LstdFlags)

	s := remote.NewSetup(cfg, logger)
	if err := s.Connect(); err != nil {
		return err
	}
	defer s.Close()

	fmt.Println()
	fmt.Printf("%s 远程 %s:%s\n", ui.Label("测试代理:"), ui.Accent(cfg.Server.Host), ui.Accent(fmt.Sprint(cfg.Tunnel.RemotePort)))
	fmt.Println()

	results := s.RunDiagnostics()

	fmt.Println(ui.Label("诊断结果:"))
	allOK := true
	for _, result := range results {
		mark := ui.SuccessMark()
		if !result.OK {
			mark = ui.ErrorMark()
			allOK = false
		}
		fmt.Printf("  %s %s: %s\n", mark, ui.Label(result.Name), result.Detail)
		if result.Hint != "" {
			fmt.Printf("    %s %s\n", ui.Warning("提示:"), result.Hint)
		}
	}

	if !allOK {
		return fmt.Errorf("%s", console.ForStderr().Error("代理连通性测试未通过"))
	}
	return nil
}
