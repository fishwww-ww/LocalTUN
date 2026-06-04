package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"localtun/internal/config"
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
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "[setup] ", log.LstdFlags)

	s := remote.NewSetup(cfg, logger)
	if err := s.Connect(); err != nil {
		return err
	}
	defer s.Close()

	fmt.Println()
	fmt.Printf("目标服务器: %s@%s:%d\n", cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("隧道端口:   远程 :%d → 本地 :%d\n", cfg.Tunnel.RemotePort, cfg.Tunnel.LocalPort)
	fmt.Println()

	if confirm("是否配置 sshd_config (AllowTcpForwarding, GatewayPorts, PermitTunnel)?") {
		if err := s.ConfigureSSHD(); err != nil {
			return fmt.Errorf("配置 sshd_config 失败: %w", err)
		}
		fmt.Println()

		if confirm("是否重启 sshd 使配置生效?") {
			if err := s.RestartSSHD(); err != nil {
				fmt.Printf("重启 sshd 失败: %v\n", err)
				fmt.Println("提示: 某些云容器或托管环境不允许重启 SSH 服务，可先继续后续配置并直接测试隧道。")
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
	fmt.Println("远程服务器配置完成!")
	fmt.Println("提示: 使用 `localtun start` 启动隧道后，在服务器上运行 `source ~/.bashrc` 或重新登录以激活代理配置。")

	return nil
}
