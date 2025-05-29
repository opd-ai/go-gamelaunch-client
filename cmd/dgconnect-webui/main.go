package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
	"github.com/opd-ai/go-gamelaunch-client/pkg/webui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
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

	// WebUI specific flags
	listenAddr   string
	tilesetPath  string
	staticPath   string
	allowOrigins []string
	pollTimeout  time.Duration
	autoLaunch   bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dgconnect-webui [user@]host",
	Short: "Web-based client for dgamelaunch SSH servers",
	Long: `dgconnect-webui is a web-based client for connecting to dgamelaunch-style SSH servers
to play terminal-based roguelike games in a web browser.

The client starts a local web server and connects to the specified SSH server,
providing a browser-based interface for playing games.

Examples:
  dgconnect-webui user@nethack.example.com
  dgconnect-webui user@server.example.com --port 2022 --listen :8080
  dgconnect-webui --config ~/.dgconnect.yaml nethack-server --tileset ./custom.yaml
  dgconnect-webui user@server.example.com --game nethack --auto-launch`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWebUIConnect,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dgconnect.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// SSH Connection flags
	rootCmd.Flags().IntVarP(&port, "port", "p", 22, "SSH port")
	rootCmd.Flags().StringVarP(&keyPath, "key", "k", "", "SSH private key path")
	rootCmd.Flags().StringVar(&password, "password", "", "SSH password (use with caution)")
	rootCmd.Flags().StringVarP(&gameName, "game", "g", "", "game to launch directly")

	// WebUI flags
	rootCmd.Flags().StringVarP(&listenAddr, "listen", "l", ":8080", "web server listen address")
	rootCmd.Flags().StringVar(&tilesetPath, "tileset", "", "path to tileset configuration file")
	rootCmd.Flags().StringVar(&staticPath, "static", "", "path to static files directory (overrides embedded)")
	rootCmd.Flags().StringSliceVar(&allowOrigins, "allow-origin", []string{}, "allowed CORS origins")
	rootCmd.Flags().DurationVar(&pollTimeout, "poll-timeout", 30*time.Second, "client polling timeout")
	rootCmd.Flags().BoolVar(&autoLaunch, "auto-launch", false, "automatically open browser")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dgconnect-webui %s (commit: %s, built: %s)\n", version, commit, date)
		},
	})

	// Serve command - for serving static content only
	rootCmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Start web server without SSH connection",
		Long: `Start the web server in demo mode without connecting to an SSH server.
Useful for testing the web interface or serving as a template.`,
		RunE: runServeOnly,
	})

	// Tileset command group
	tilesetCmd := &cobra.Command{
		Use:   "tileset",
		Short: "Tileset management commands",
	}

	tilesetCmd.AddCommand(&cobra.Command{
		Use:   "create [output-file]",
		Short: "Create example tileset configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCreateTileset,
	})

	tilesetCmd.AddCommand(&cobra.Command{
		Use:   "validate [tileset-file]",
		Short: "Validate tileset configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  runValidateTileset,
	})

	rootCmd.AddCommand(tilesetCmd)
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

func runWebUIConnect(cmd *cobra.Command, args []string) error {
	var host, user string
	var actualPort int

	// Parse connection string or use config
	if len(args) > 0 {
		if err := parseConnectionString(args[0], &user, &host); err != nil {
			return err
		}
		actualPort = port // Use command line port
	} else {
		// Try to use default server from config
		defaultServer := viper.GetString("default_server")
		if defaultServer == "" {
			return fmt.Errorf("no server specified and no default_server in config")
		}

		serverConfig, err := getServerConfig(defaultServer)
		if err != nil {
			return err
		}

		host = serverConfig.Host
		user = serverConfig.Username
		actualPort = serverConfig.Port
		if actualPort == 0 {
			actualPort = 22
		}

		// Use config tileset if not specified on command line
		if tilesetPath == "" && serverConfig.TilesetPath != "" {
			tilesetPath = serverConfig.TilesetPath
		}
	}

	// Validate required parameters
	if host == "" {
		return fmt.Errorf("host is required")
	}
	if user == "" {
		return fmt.Errorf("username is required")
	}

	// Create dgclient
	clientConfig := dgclient.DefaultClientConfig()
	clientConfig.Debug = debug

	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: getHostKeyCallback(),
		Timeout:         clientConfig.ConnectTimeout,
	}
	clientConfig.SSHConfig = sshConfig

	client := dgclient.NewClient(clientConfig)
	defer client.Close()

	// Create web view
	viewOpts := dgclient.DefaultViewOptions()
	view, err := webui.NewWebView(viewOpts)
	if err != nil {
		return fmt.Errorf("failed to create web view: %w", err)
	}

	if err := client.SetView(view); err != nil {
		return fmt.Errorf("failed to set view: %w", err)
	}

	// Create WebUI options
	webuiOpts := webui.WebUIOptions{
		View:         view,
		ListenAddr:   listenAddr,
		TilesetPath:  tilesetPath,
		StaticPath:   staticPath,
		AllowOrigins: allowOrigins,
		PollTimeout:  pollTimeout,
	}

	// Create WebUI server
	webuiServer, err := webui.NewWebUI(webuiOpts)
	if err != nil {
		return fmt.Errorf("failed to create WebUI server: %w", err)
	}

	// Get authentication method
	auth, err := getAuthMethod(user, host)
	if err != nil {
		return fmt.Errorf("failed to get authentication method: %w", err)
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Start web server
	go func() {
		fmt.Printf("Starting WebUI server on %s...\n", listenAddr)
		if err := webuiServer.StartWithContext(ctx, listenAddr); err != nil {
			fmt.Printf("WebUI server error: %v\n", err)
			cancel()
		}
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect to SSH server
	fmt.Printf("Connecting to %s@%s:%d...\n", user, host, actualPort)
	if err := client.Connect(host, actualPort, auth); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	fmt.Println("Connected successfully!")

	// Launch game if specified
	if gameName != "" {
		if err := client.SelectGame(gameName); err != nil {
			fmt.Printf("Warning: failed to select game %s: %v\n", gameName, err)
		}
	}

	// Show access information
	showAccessInfo(listenAddr)

	// Auto-launch browser if requested
	if autoLaunch {
		if err := openBrowser(getWebURL(listenAddr)); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}

	// Start SSH client
	go func() {
		if err := client.Run(ctx); err != nil {
			fmt.Printf("SSH client error: %v\n", err)
			cancel()
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	fmt.Println("Shutting down...")
	return nil
}

func runServeOnly(cmd *cobra.Command, args []string) error {
	webuiOpts := webui.WebUIOptions{
		ListenAddr:   listenAddr,
		TilesetPath:  tilesetPath,
		StaticPath:   staticPath,
		AllowOrigins: allowOrigins,
		PollTimeout:  pollTimeout,
	}

	webuiServer, err := webui.NewWebUI(webuiOpts)
	if err != nil {
		return fmt.Errorf("failed to create WebUI server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Printf("Starting WebUI server on %s (demo mode)...\n", listenAddr)
	showAccessInfo(listenAddr)

	if autoLaunch {
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := openBrowser(getWebURL(listenAddr)); err != nil {
				fmt.Printf("Failed to open browser: %v\n", err)
			}
		}()
	}

	return webuiServer.StartWithContext(ctx, listenAddr)
}

func runCreateTileset(cmd *cobra.Command, args []string) error {
	var outputPath string
	if len(args) > 0 {
		outputPath = args[0]
	} else {
		outputPath = "tileset.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("File %s already exists. Overwrite? (y/n): ", outputPath)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Create default tileset
	tileset := webui.DefaultTilesetConfig()

	if err := webui.SaveTilesetConfig(tileset, outputPath); err != nil {
		return fmt.Errorf("failed to save tileset: %w", err)
	}

	fmt.Printf("Example tileset configuration created: %s\n", outputPath)
	fmt.Println("Edit the file to customize character mappings and add your tileset image.")
	return nil
}

func runValidateTileset(cmd *cobra.Command, args []string) error {
	tilesetPath := args[0]

	tileset, err := webui.LoadTilesetConfig(tilesetPath)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("✓ Tileset configuration is valid\n")
	fmt.Printf("  Name: %s\n", tileset.Name)
	fmt.Printf("  Version: %s\n", tileset.Version)
	fmt.Printf("  Tile size: %dx%d\n", tileset.TileWidth, tileset.TileHeight)
	fmt.Printf("  Mappings: %d\n", len(tileset.Mappings))
	fmt.Printf("  Special tiles: %d\n", len(tileset.SpecialTiles))

	if tileset.GetImageData() != nil {
		tilesX, tilesY := tileset.GetTileCount()
		fmt.Printf("  Image loaded: %dx%d tiles\n", tilesX, tilesY)
	} else {
		fmt.Printf("  Image: not loaded\n")
	}

	return nil
}

// Helper functions from dgconnect

func parseConnectionString(conn string, user, host *string) error {
	parts := strings.Split(conn, "@")
	if len(parts) == 2 {
		*user = parts[0]
		*host = parts[1]
	} else if len(parts) == 1 {
		*host = parts[0]
		*user = os.Getenv("USER")
		if *user == "" {
			return fmt.Errorf("no username specified and USER environment variable not set")
		}
	} else {
		return fmt.Errorf("invalid connection string: %s", conn)
	}
	return nil
}

func getAuthMethod(user, host string) (dgclient.AuthMethod, error) {
	if password != "" {
		return dgclient.NewPasswordAuth(password), nil
	}

	if keyPath != "" {
		return dgclient.NewKeyAuth(keyPath, ""), nil
	}

	// Try SSH agent
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return dgclient.NewAgentAuth(), nil
	}

	// Try default key locations
	home, _ := os.UserHomeDir()
	defaultKeys := []string{
		fmt.Sprintf("%s/.ssh/id_rsa", home),
		fmt.Sprintf("%s/.ssh/id_ed25519", home),
		fmt.Sprintf("%s/.ssh/id_ecdsa", home),
	}

	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			return dgclient.NewKeyAuth(keyPath, ""), nil
		}
	}

	// Fall back to password prompt
	fmt.Printf("Password for %s@%s: ", user, host)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	return dgclient.NewPasswordAuth(string(passwordBytes)), nil
}

func getHostKeyCallback() ssh.HostKeyCallback {
	home, err := os.UserHomeDir()
	if err != nil {
		return ssh.InsecureIgnoreHostKey()
	}

	knownHostsPath := fmt.Sprintf("%s/.ssh/known_hosts", home)
	if _, err := os.Stat(knownHostsPath); err != nil {
		return ssh.InsecureIgnoreHostKey()
	}

	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return ssh.InsecureIgnoreHostKey()
	}

	return hostKeyCallback
}

// WebUI specific helpers

func showAccessInfo(listenAddr string) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("🌐 WebUI Server Started\n")
	fmt.Println(strings.Repeat("=", 50))

	url := getWebURL(listenAddr)
	fmt.Printf("📱 Web Interface: %s\n", url)

	if strings.HasPrefix(listenAddr, ":") {
		// Show all local IPs if binding to all interfaces
		if ips := getLocalIPs(); len(ips) > 0 {
			fmt.Println("📍 Available on:")
			for _, ip := range ips {
				if ip != "127.0.0.1" {
					port := strings.TrimPrefix(listenAddr, ":")
					fmt.Printf("   http://%s:%s\n", ip, port)
				}
			}
		}
	}

	fmt.Println("\n💡 Instructions:")
	fmt.Println("   • Open the web interface in your browser")
	fmt.Println("   • Click on the game canvas to focus input")
	fmt.Println("   • Use keyboard for game controls")
	fmt.Println("   • Press Ctrl+C to stop the server")
	fmt.Println(strings.Repeat("=", 50) + "\n")
}

func getWebURL(listenAddr string) string {
	if strings.HasPrefix(listenAddr, ":") {
		return fmt.Sprintf("http://localhost%s", listenAddr)
	}
	if !strings.Contains(listenAddr, "://") {
		return fmt.Sprintf("http://%s", listenAddr)
	}
	return listenAddr
}

func getLocalIPs() []string {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case commandExists("xdg-open"):
		cmd = "xdg-open"
		args = []string{url}
	case commandExists("open"):
		cmd = "open"
		args = []string{url}
	case commandExists("cmd"):
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("no suitable browser launcher found")
	}

	return exec.Command(cmd, args...).Start()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Configuration structures (simplified from main dgconnect)

type ServerConfig struct {
	Host        string     `yaml:"host"`
	Port        int        `yaml:"port"`
	Username    string     `yaml:"username"`
	Auth        AuthConfig `yaml:"auth"`
	DefaultGame string     `yaml:"default_game,omitempty"`
	TilesetPath string     `yaml:"tileset_path,omitempty"`
}

type AuthConfig struct {
	Method     string `yaml:"method"`
	KeyPath    string `yaml:"key_path,omitempty"`
	Passphrase string `yaml:"passphrase,omitempty"`
}

func getServerConfig(name string) (*ServerConfig, error) {
	serverKey := fmt.Sprintf("servers.%s", name)
	if !viper.IsSet(serverKey) {
		return nil, fmt.Errorf("server '%s' not found in configuration", name)
	}

	var server ServerConfig
	if err := viper.UnmarshalKey(serverKey, &server); err != nil {
		return nil, fmt.Errorf("failed to parse server configuration: %w", err)
	}

	if server.Port == 0 {
		server.Port = 22
	}

	return &server, nil
}
