package cli

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dgrpc"
	"github.com/streamingfast/dlauncher/launcher"
)

func init() {
	dgrpc.Verbosity = 2
}

func setupCmd(cmd *cobra.Command) error {
	cmd.SilenceUsage = true

	cmds := extractCmd(cmd)
	subCommand := cmds[len(cmds)-1]

	forceConfigOn := []*cobra.Command{StartCmd}
	logToFileOn := []*cobra.Command{StartCmd}

	if configFile := viper.GetString("global-config-file"); configFile != "" {
		exists, err := fileExists(configFile)
		if err != nil {
			return fmt.Errorf("unable to check if config file exists: %w", err)
		}

		if !exists && isMatchingCommand(cmds, forceConfigOn) {
			return fmt.Errorf("config file %q not found. Did you 'fireaptos init'?", configFile)
		}

		if exists {
			if err := launcher.LoadConfigFile(configFile); err != nil {
				return fmt.Errorf("unable to read config file %q: %w", configFile, err)
			}
		}
	}

	subconf := launcher.Config[subCommand]
	if subconf != nil {
		for k, v := range subconf.Flags {
			validFlag := false
			if _, ok := allFlags["global-"+k]; ok {
				viper.SetDefault("global-"+k, v)
				validFlag = true
			}
			if _, ok := allFlags[k]; ok {
				viper.SetDefault(k, v)
				validFlag = true
			}
			if !validFlag {
				return fmt.Errorf("invalid flag %s in config file under command %s", k, subCommand)
			}
		}
	}

	launcher.SetupLogger(rootLog, &launcher.LoggingOptions{
		WorkingDir:    viper.GetString("global-data-dir"),
		Verbosity:     viper.GetInt("global-log-verbosity"),
		LogFormat:     viper.GetString("global-log-format"),
		LogToFile:     isMatchingCommand(cmds, logToFileOn) && viper.GetBool("global-log-to-file"),
		LogListenAddr: viper.GetString("global-log-level-switcher-listen-addr"),
	})
	launcher.SetupTracing("fireaptos")
	launcher.SetupAnalyticsMetrics(rootLog, viper.GetString("global-metrics-listen-addr"), viper.GetString("global-pprof-listen-addr"))

	return nil
}

func isMatchingCommand(cmds []string, runSetupOn []*cobra.Command) bool {
	for _, c := range runSetupOn {
		baseChunks := extractCmd(c)
		if strings.Join(cmds, ".") == strings.Join(baseChunks, ".") {
			return true
		}
	}
	return false
}

func extractCmd(cmd *cobra.Command) []string {
	cmds := []string{}
	for {
		if cmd == nil {
			break
		}
		cmds = append(cmds, cmd.Use)
		cmd = cmd.Parent()
	}

	out := make([]string, len(cmds))

	for itr, v := range cmds {
		newIndex := len(cmds) - 1 - itr
		out[newIndex] = v
	}
	return out
}

func fileExists(file string) (bool, error) {
	stat, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return !stat.IsDir(), nil
}
