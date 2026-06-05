package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
	"localtun/internal/remote"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "配置远程服务器 (sshd_config, .bashrc)",
	Long:  `通过 SSH 连接远程服务器，自动配置 sshd_config 和 .bashrc 中的代理设置。`,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func confirm(prompt string) bool {
	ui := console.ForStdout()
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s %s: ", prompt, ui.Muted("[y/N]"))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	ui := console.ForStdout()
	logger := log.New(os.Stdout, ui.Prefix("setup"), log.LstdFlags)

	s := remote.NewSetup(cfg, logger)
	if err := s.Connect(); err != nil {
		return err
	}
	defer s.Close()

	fmt.Println()
	fmt.Printf("%s %s@%s:%s\n", ui.Label("目标服务器:"), ui.Info(cfg.Server.User), ui.Accent(cfg.Server.Host), ui.Accent(fmt.Sprint(cfg.Server.Port)))
	fmt.Printf("%s   远程 %s → 本地 %s\n", ui.Label("隧道端口:"), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.LocalPort)))
	fmt.Println()

	if confirm("是否配置 sshd_config (AllowTcpForwarding, GatewayPorts, PermitTunnel)?") {
		if err := s.ConfigureSSHD(); err != nil {
			return fmt.Errorf("配置 sshd_config 失败: %w", err)
		}
		fmt.Println()

		if confirm("是否重启 sshd 使配置生效?") {
			if err := s.RestartSSHD(); err != nil {
				fmt.Printf("%s %v\n", ui.Error("重启 sshd 失败:"), err)
				fmt.Printf("%s 某些云容器或托管环境不允许重启 SSH 服务，可先继续后续配置并直接测试隧道。\n", ui.Warning("提示:"))
				if !confirm("是否继续配置 .bashrc?") {
					return nil
				}
			}
		}
	}

	fmt.Println()

	if confirm("是否配置 .bashrc (代理环境变量和辅助函数)?") {
		if err := s.ConfigureBashRC(); err != nil {
			return fmt.Errorf("配置 .bashrc 失败: %w", err)
		}
	}

	fmt.Println()
	fmt.Printf("%s 远程服务器配置完成!\n", ui.SuccessMark())
	fmt.Printf("%s 使用 %s 启动隧道后，在服务器上运行 %s 或重新登录以激活代理配置。\n", ui.Warning("提示:"), ui.Info("`localtun start`"), ui.Info("`source ~/.bashrc`"))

	return nil
}
