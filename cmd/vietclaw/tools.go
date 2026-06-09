package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"vietclaw/internal/config"
)

const toolsNetworkPolicy = "network-policy"

func runTools(args []string) error {
	if len(args) == 0 {
		return runNetworkPolicyTUI()
	}
	switch args[0] {
	case toolsNetworkPolicy, "net":
		return runNetworkPolicy(args[1:])
	default:
		return fmt.Errorf("unknown tools command %q", args[0])
	}
}

func runNetworkPolicy(args []string) error {
	if len(args) == 0 || args[0] == "tui" {
		return runNetworkPolicyTUI()
	}
	switch args[0] {
	case "show":
		_, cfg, err := loadOrCreateConfig()
		if err != nil {
			return err
		}
		printNetworkPolicy(cfg)
		return nil
	case "allow", "deny":
		return runNetworkPolicyListCommand(config.NetworkPolicyList(args[0]), args[1:])
	case "enable":
		return updateNetworkPolicy(func(cfg config.Config) config.Config {
			return config.SetShellNetworkPolicyEnabled(cfg, true)
		})
	case "disable":
		return updateNetworkPolicy(func(cfg config.Config) config.Config {
			return config.SetShellNetworkPolicyEnabled(cfg, false)
		})
	case "restrict":
		if len(args) < 2 {
			return fmt.Errorf("usage: vietclaw tools network-policy restrict <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		return updateNetworkPolicy(func(cfg config.Config) config.Config {
			return config.SetShellNetworkPolicyRestrictToAllow(cfg, enabled)
		})
	case "deny-private":
		if len(args) < 2 {
			return fmt.Errorf("usage: vietclaw tools network-policy deny-private <on|off>")
		}
		enabled, err := parseOnOff(args[1])
		if err != nil {
			return err
		}
		return updateNetworkPolicy(func(cfg config.Config) config.Config {
			return config.SetShellNetworkPolicyDenyPrivate(cfg, enabled)
		})
	default:
		return fmt.Errorf("unknown network-policy command %q", args[0])
	}
}

func runNetworkPolicyListCommand(list config.NetworkPolicyList, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: vietclaw tools network-policy %s <add|remove> <host>", list)
	}
	action := args[0]
	host := args[1]
	switch action {
	case "add":
		return updateNetworkPolicyWithError(func(cfg config.Config) (config.Config, error) {
			return config.AddShellNetworkHost(cfg, list, host)
		})
	case "remove", "rm":
		return updateNetworkPolicyWithError(func(cfg config.Config) (config.Config, error) {
			return config.RemoveShellNetworkHost(cfg, list, host)
		})
	default:
		return fmt.Errorf("unknown %s command %q", list, action)
	}
}

func runNetworkPolicyTUI() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		paths, cfg, err := loadOrCreateConfig()
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("VietClaw Network Policy")
		fmt.Println(strings.Repeat("-", 32))
		printNetworkPolicy(cfg)
		fmt.Println()
		fmt.Println("1) Add allow host")
		fmt.Println("2) Remove allow host")
		fmt.Println("3) Add deny host")
		fmt.Println("4) Remove deny host")
		fmt.Println("5) Toggle allowlist-only mode")
		fmt.Println("6) Toggle deny private/link-local IP")
		fmt.Println("7) Toggle policy enabled")
		fmt.Println("0) Exit")
		fmt.Print("> ")
		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		choice = strings.TrimSpace(choice)
		if choice == "0" || strings.EqualFold(choice, "q") {
			return nil
		}
		updated := cfg
		switch choice {
		case "1":
			host, err := promptHost(reader, "Allow host")
			if err != nil {
				return err
			}
			updated, err = config.AddShellNetworkHost(cfg, config.NetworkPolicyAllow, host)
			if err != nil {
				return err
			}
		case "2":
			host, err := promptHost(reader, "Remove allow host")
			if err != nil {
				return err
			}
			updated, err = config.RemoveShellNetworkHost(cfg, config.NetworkPolicyAllow, host)
			if err != nil {
				return err
			}
		case "3":
			host, err := promptHost(reader, "Deny host")
			if err != nil {
				return err
			}
			updated, err = config.AddShellNetworkHost(cfg, config.NetworkPolicyDeny, host)
			if err != nil {
				return err
			}
		case "4":
			host, err := promptHost(reader, "Remove deny host")
			if err != nil {
				return err
			}
			updated, err = config.RemoveShellNetworkHost(cfg, config.NetworkPolicyDeny, host)
			if err != nil {
				return err
			}
		case "5":
			updated = config.SetShellNetworkPolicyRestrictToAllow(cfg, !cfg.Tools.Shell.NetworkPolicy.RestrictToAllowHosts)
		case "6":
			updated = config.SetShellNetworkPolicyDenyPrivate(cfg, !cfg.Tools.Shell.NetworkPolicy.DenyPrivate)
		case "7":
			updated = config.SetShellNetworkPolicyEnabled(cfg, !cfg.Tools.Shell.NetworkPolicy.Enabled)
		default:
			fmt.Println("[warn] unknown option")
			continue
		}
		if err := config.Save(paths.ConfigFile, updated); err != nil {
			return err
		}
		fmt.Println("[ok] saved")
	}
}

func printNetworkPolicy(cfg config.Config) {
	policy := cfg.Tools.Shell.NetworkPolicy
	fmt.Printf("enabled: %t\n", policy.Enabled)
	fmt.Printf("allowlist_only: %t\n", policy.RestrictToAllowHosts)
	fmt.Printf("deny_private: %t\n", policy.DenyPrivate)
	fmt.Printf("allow_hosts: %s\n", formatHosts(policy.AllowHosts))
	fmt.Printf("deny_hosts: %s\n", formatHosts(policy.DenyHosts))
}

func formatHosts(hosts []string) string {
	if len(hosts) == 0 {
		return "(empty)"
	}
	return strings.Join(hosts, ", ")
}

func updateNetworkPolicy(update func(config.Config) config.Config) error {
	return updateNetworkPolicyWithError(func(cfg config.Config) (config.Config, error) {
		return update(cfg), nil
	})
}

func updateNetworkPolicyWithError(update func(config.Config) (config.Config, error)) error {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	cfg, err = update(cfg)
	if err != nil {
		return err
	}
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return err
	}
	printNetworkPolicy(cfg)
	return nil
}

func parseOnOff(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes", "y":
		return true, nil
	case "off", "false", "0", "no", "n":
		return false, nil
	default:
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed, nil
		}
		return false, fmt.Errorf("expected on or off, got %q", value)
	}
}

func promptHost(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	return readLine(reader)
}

func readLine(reader *bufio.Reader) (string, error) {
	text, err := reader.ReadString('\n')
	if err != nil {
		return strings.TrimSpace(text), err
	}
	return strings.TrimSpace(text), nil
}
