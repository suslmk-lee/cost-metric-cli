package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kpaas-cost",
	Short: "Kubernetes CSP cost analysis tool",
	Long: `K-PaaS Cost is a CLI tool that analyzes Kubernetes cluster costs
across different Cloud Service Providers (CSP).

It automatically detects your current Kubernetes context,
identifies the CSP, analyzes node specifications, and
calculates the cost based on real-time pricing information.`,
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveMode()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kpaas-cost.yaml)")
	rootCmd.PersistentFlags().StringP("context", "c", "", "Kubernetes context to use")
	rootCmd.PersistentFlags().StringP("duration", "d", "1h", "Duration for cost calculation (1h, 1d, 1m, 1y)")
	rootCmd.PersistentFlags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output file path")

	viper.BindPFlag("context", rootCmd.PersistentFlags().Lookup("context"))
	viper.BindPFlag("duration", rootCmd.PersistentFlags().Lookup("duration"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kpaas-cost")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
