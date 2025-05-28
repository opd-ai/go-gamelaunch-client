package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Configuration
	cfgFile string

	// Command flags
	port     int
	keyPath  string
	password string
	gameName string
	debug    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dgconnect [user@]host",
	Short: "Connect to dgamelaunch SSH servers",
	Long: `dgconnect is a client for connecting to dgamelaunch-style SSH servers
to play terminal-based roguelike games remotely.

Examples:
  dgconnect user@nethack.example.com
  dgconnect user@server.example.com --port 2022 --key ~/.ssh/id_rsa
  dgconnect --config ~/.dgconnect.yaml nethack-server
  dgconnect user@server.example.com --game nethack`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConnect,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dgconnect.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Connection flags
	rootCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
	rootCmd.Flags().StringVarP(&keyPath, "key", "k", "", "SSH private key path")
	rootCmd.Flags().StringVar(&password, "password", "", "SSH password (use with caution)")
	rootCmd.Flags().StringVarP(&gameName, "game", "g", "", "game to launch directly")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dgconnect %s (commit: %s, built: %s)\n", version, commit, date)
		},
	})
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".dgconnect")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if debug {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}
