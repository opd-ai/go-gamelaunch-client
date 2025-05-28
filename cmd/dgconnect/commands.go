package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/opd-ai/go-gamelaunch-client/pkg/dgclient"
	"github.com/opd-ai/go-gamelaunch-client/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func runConnect(cmd *cobra.Command, args []string) error {
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

		serverConfig, err := GetServerConfig(defaultServer)
		if err != nil {
			return err
		}

		host = serverConfig.Host
		user = serverConfig.Username
		actualPort = serverConfig.Port
		if actualPort == 0 {
			actualPort = 22
		}
	}

	// Validate required parameters
	if host == "" {
		return fmt.Errorf("host is required")
	}
	if user == "" {
		return fmt.Errorf("username is required")
	}

	// Create client configuration
	clientConfig := dgclient.DefaultClientConfig()
	clientConfig.Debug = debug

	// Set up SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: getHostKeyCallback(),
		Timeout:         clientConfig.ConnectTimeout,
	}
	clientConfig.SSHConfig = sshConfig

	// Create client
	client := dgclient.NewClient(clientConfig)
	defer client.Close()

	// Set up view
	viewOpts := dgclient.DefaultViewOptions()
	view, err := tui.NewTerminalView(viewOpts)
	if err != nil {
		return fmt.Errorf("failed to create terminal view: %w", err)
	}

	if err := client.SetView(view); err != nil {
		return fmt.Errorf("failed to set view: %w", err)
	}

	// Get authentication method
	auth, err := getAuthMethod(user, host)
	if err != nil {
		return fmt.Errorf("failed to get authentication method: %w", err)
	}

	// Connect
	fmt.Printf("Connecting to %s@%s:%d...\n", user, host, actualPort)
	if err := client.Connect(host, actualPort, auth); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	fmt.Println("Connected successfully!")

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, disconnecting...")
		cancel()
	}()

	// Launch game if specified
	if gameName != "" {
		if err := client.SelectGame(gameName); err != nil {
			fmt.Printf("Warning: failed to select game %s: %v\n", gameName, err)
		}
	}

	// Run the client
	if err := client.Run(ctx); err != nil {
		return fmt.Errorf("client error: %w", err)
	}

	return nil
}

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
	// Priority: command line flag > config > SSH agent > default keys > password prompt

	if password != "" {
		return dgclient.NewPasswordAuth(password), nil
	}

	if keyPath != "" {
		return dgclient.NewKeyAuth(keyPath, ""), nil
	}

	// Check config for auth method
	defaultServer := viper.GetString("default_server")
	if defaultServer != "" {
		serverConfig, err := GetServerConfig(defaultServer)
		if err == nil {
			switch serverConfig.Auth.Method {
			case "key":
				if serverConfig.Auth.KeyPath != "" {
					return dgclient.NewKeyAuth(expandPath(serverConfig.Auth.KeyPath), serverConfig.Auth.Passphrase), nil
				}
			case "password":
				// Will fall through to password prompt
			case "agent":
				if os.Getenv("SSH_AUTH_SOCK") != "" {
					return dgclient.NewAgentAuth(), nil
				}
			}
		}
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
	// Try to use known_hosts file first
	home, err := os.UserHomeDir()
	if err == nil {
		knownHostsPath := fmt.Sprintf("%s/.ssh/known_hosts", home)
		if _, err := os.Stat(knownHostsPath); err == nil {
			// In a production version, you'd use knownhosts.New(knownHostsPath)
			// For now, we'll use an insecure callback with warning
		}
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if debug {
			fmt.Printf("Warning: Accepting host key for %s\n", hostname)
			fmt.Printf("Fingerprint: %s\n", ssh.FingerprintSHA256(key))
		}
		return nil
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return strings.Replace(path, "~", home, 1)
		}
	}
	return path
}
