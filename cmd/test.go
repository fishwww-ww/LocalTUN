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
	testCmd.Flags().StringArrayVarP(&selectedServers, "server", "s", nil, "只处理指定服务器，可重复传入")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}
	profiles, err := selectProfiles(cfg, selectedServers)
	if err != nil {
		return err
	}

	ui := console.ForStdout()
	logger := log.New(os.Stdout, ui.Prefix("test"), log.LstdFlags)

	allOK := true
	for _, profile := range profiles {
		ok, err := runTestProfile(profile, ui, logger)
		if err != nil {
			return err
		}
		if !ok {
			allOK = false
		}
	}

	if !allOK {
		return fmt.Errorf("%s", console.ForStderr().Error("代理连通性测试未通过"))
	}
	return nil
}

func runTestProfile(profile selectedProfile, ui console.Styler, logger *log.Logger) (bool, error) {
	s := remote.NewSetup(profile.Runtime, logger)
	if err := s.Connect(); err != nil {
		return false, fmt.Errorf("[%s] %w", profile.Name, err)
	}
	defer s.Close()

	fmt.Println()
	fmt.Printf("%s %s\n", ui.Label("服务器名称:"), ui.Info(profile.Name))
	fmt.Printf("%s 远程 %s:%s\n", ui.Label("测试代理:"), ui.Accent(profile.Runtime.Server.Host), ui.Accent(fmt.Sprint(profile.Runtime.Tunnel.RemotePort)))
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

	return allOK, nil
}
