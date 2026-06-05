package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "管理服务器配置",
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出服务器配置",
	RunE:  runServerList,
}

var serverAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "添加或覆盖服务器配置",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runServerAdd,
}

var serverRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "删除服务器配置",
	Args:  cobra.ExactArgs(1),
	RunE:  runServerRemove,
}

func init() {
	serverCmd.AddCommand(serverListCmd, serverAddCmd, serverRemoveCmd)
	rootCmd.AddCommand(serverCmd)
}

func runServerList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	ui := console.ForStdout()
	for _, name := range sortedProfileNames(cfg) {
		profile := cfg.Servers[name]
		fmt.Printf("%s %s\n", ui.Label("服务器:"), ui.Info(name))
		fmt.Printf("  %s     %s@%s:%s\n", ui.Label("SSH:"), ui.Info(profile.User), ui.Accent(profile.Host), ui.Accent(fmt.Sprint(profile.Port)))
		fmt.Printf("  %s   远程 %s → 本地 %s\n", ui.Label("隧道:"), ui.Accent(fmt.Sprintf(":%d", profile.RemotePort)), ui.Accent(fmt.Sprintf(":%d", profile.LocalPort)))
	}
	return nil
}

func runServerAdd(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfigForEdit()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	name := ""
	if len(args) == 1 {
		name = args[0]
		if err := validateProfileName(name); err != nil {
			return err
		}
	} else {
		name = promptServerNameForEdit(reader)
	}

	ui := console.ForStdout()
	if _, exists := cfg.Servers[name]; exists && !confirm(fmt.Sprintf("服务器 %s 已存在，是否覆盖?", ui.Accent(name))) {
		fmt.Println(ui.Muted("已取消"))
		return nil
	}

	cfg.Servers[name] = promptServerProfile(reader, config.DefaultConfig())
	if err := cfg.Save(configPathForEdit()); err != nil {
		return err
	}
	fmt.Printf("%s 服务器 %s 已保存\n", ui.SuccessMark(), ui.Info(name))
	return nil
}

func runServerRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	name := args[0]
	if _, exists := cfg.Servers[name]; !exists {
		return fmt.Errorf("未知服务器: %s。可用服务器: %s", name, strings.Join(sortedProfileNames(cfg), ", "))
	}

	ui := console.ForStdout()
	if !confirm(fmt.Sprintf("是否删除服务器 %s?", ui.Accent(name))) {
		fmt.Println(ui.Muted("已取消"))
		return nil
	}

	delete(cfg.Servers, name)
	if len(cfg.Servers) == 0 {
		return fmt.Errorf("至少需要保留一个服务器配置")
	}
	if err := cfg.Save(configPathForEdit()); err != nil {
		return err
	}
	fmt.Printf("%s 服务器 %s 已删除\n", ui.SuccessMark(), ui.Info(name))
	return nil
}

func loadConfigForEdit() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err == nil {
		return cfg, nil
	}
	if _, statErr := os.Stat(configPathForEdit()); statErr == nil {
		return nil, err
	}
	cfg = config.DefaultConfig()
	return cfg, nil
}

func configPathForEdit() string {
	if cfgFile != "" {
		return cfgFile
	}
	return config.DefaultConfigPath()
}

func promptServerNameForEdit(reader *bufio.Reader) string {
	ui := console.ForStdout()
	for {
		name := prompt(reader, "服务器名称", "")
		if err := validateProfileName(name); err != nil {
			fmt.Printf("  %s %s\n", ui.WarningMark(), ui.Warning(err.Error()))
			continue
		}
		return name
	}
}
