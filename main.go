// Rancher Log Viewer v1.0.0
// Autor: Luiz Ricardo Z. Calmon

package main

import (
	"fmt"
	"os"

	"rancher-logs/api"
	"rancher-logs/config"
	"rancher-logs/logger"
	"rancher-logs/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			PaddingLeft(1)
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			PaddingLeft(1)
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			PaddingLeft(1)
	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			PaddingLeft(1)
)

func main() {
	// Handle --about / --version flags
	if len(os.Args) > 1 && (os.Args[1] == "--about" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		showAbout()
		return
	}

	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize debug.log: %v\n", err)
	}
	defer logger.Close()

	logger.Info("=== Rancher Log Viewer started ===")

	// Step 1: Config
	cfg, err := ensureConfig()
	if err != nil {
		printError("Configuration error", err)
		os.Exit(1)
	}
	if cfg == nil {
		os.Exit(0)
	}

	client := api.NewClient(cfg)

	// Outer loop: cluster selection
	for {
		// Step 2: Select cluster
		clusterID, clusterName, err := selectCluster(client)
		if err != nil {
			printError("Failed to load clusters", err)
			os.Exit(1)
		}
		if clusterID == "" {
			fmt.Println(infoStyle.Render("\nExiting. Goodbye!"))
			os.Exit(0)
		}
		if clusterID == "about" {
			continue
		}
		client.SetCluster(clusterID)

		// Middle loop: namespace selection
		nsLoop := true
		for nsLoop {
			namespace, err := selectNamespace(client)
			if err != nil {
				printError("Failed to load namespaces", err)
				os.Exit(1)
			}
			if namespace == "" {
				// ESC → back to cluster selection
				nsLoop = false
				break
			}

			// Inner loop: pod selection
			for {
				podName, err := selectPod(client, namespace, clusterName)
				if err != nil {
					printWarning("Failed to load pods", err)
					break // back to namespace selection
				}
				if podName == "" {
					// ESC → back to namespace selection
					break
				}

				// Step 5: Stream logs
				if err := streamLogs(client, namespace, podName, cfg); err != nil {
					printWarning("Log streaming error", err)
					// back to pod selection
				}
			}
		}
	}
}

func ensureConfig() (*config.Config, error) {
	if config.Exists() {
		logger.Info("Loading existing config.json")
		return config.Load()
	}

	logger.Info("No config.json found, starting setup wizard")

	setupModel := ui.NewSetupModel()
	p := tea.NewProgram(setupModel)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("setup UI error: %w", err)
	}

	m, ok := finalModel.(ui.SetupModel)
	if !ok || !m.Done() {
		return nil, nil
	}

	cfg := m.Config
	if cfg.RancherURL == "" || cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("all fields (URL, Username, Password) are required")
	}
	cfg.MaxLines = config.DefaultMaxLines

	if err := config.Save(cfg); err != nil {
		return nil, err
	}

	fmt.Println(successStyle.Render("✓ config.json saved successfully!"))
	logger.Info("config.json saved")

	return cfg, nil
}

func selectCluster(client *api.Client) (string, string, error) {
	fmt.Println(infoStyle.Render("\n⏳ Loading clusters..."))

	clusters, err := client.GetClusters()
	if err != nil {
		return "", "", err
	}

	if len(clusters) == 0 {
		return "", "", fmt.Errorf("no clusters found")
	}

	logger.Info(fmt.Sprintf("Found %d clusters", len(clusters)))

	clusterModel := ui.NewClusterModel(clusters)
	p := tea.NewProgram(clusterModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", "", fmt.Errorf("cluster UI error: %w", err)
	}

	m, ok := finalModel.(ui.ClusterModel)
	if !ok || m.Quit() {
		return "", "", nil
	}

	if m.ShowAbout() {
		showAbout()
		return "about", "", nil
	}

	logger.Info(fmt.Sprintf("Selected cluster: %s (%s)", m.SelectedName(), m.SelectedID()))
	return m.SelectedID(), m.SelectedName(), nil
}

func selectNamespace(client *api.Client) (string, error) {
	fmt.Println(infoStyle.Render("\n⏳ Loading namespaces..."))

	namespaces, err := client.GetNamespaces()
	if err != nil {
		return "", err
	}

	if len(namespaces) == 0 {
		return "", fmt.Errorf("no namespaces found in cluster")
	}

	logger.Info(fmt.Sprintf("Found %d namespaces", len(namespaces)))

	nsModel := ui.NewNamespaceModel(namespaces)
	p := tea.NewProgram(nsModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("namespace UI error: %w", err)
	}

	m, ok := finalModel.(ui.NamespaceModel)
	if !ok || m.Quit() {
		return "", nil
	}

	selected := m.Selected()
	logger.Info(fmt.Sprintf("Selected namespace: %s", selected))
	return selected, nil
}

func selectPod(client *api.Client, namespace, clusterName string) (string, error) {
	fmt.Println(infoStyle.Render(fmt.Sprintf("\n⏳ Loading pods in [%s]...", namespace)))

	pods, err := client.GetPods(namespace)
	if err != nil {
		return "", err
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found in namespace '%s' — select another namespace", namespace)
	}

	logger.Info(fmt.Sprintf("Found %d pods in %s", len(pods), namespace))

	podModel := ui.NewPodModel(pods, namespace, clusterName)
	p := tea.NewProgram(podModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("pod UI error: %w", err)
	}

	m, ok := finalModel.(ui.PodModel)
	if !ok || m.Quit() {
		return "", nil
	}

	selected := m.Selected()
	logger.Info(fmt.Sprintf("Selected pod: %s", selected))
	return selected, nil
}

func streamLogs(client *api.Client, namespace, podName string, cfg *config.Config) error {
	fmt.Println(infoStyle.Render(fmt.Sprintf("\n⏳ Connecting to logs of [%s] (last %d lines)...", podName, cfg.MaxLines)))

	reader, err := client.StreamLogs(namespace, podName, cfg.MaxLines)
	if err != nil {
		return err
	}

	logModel := ui.NewLogModel(podName, namespace, reader, cfg, client)
	p := tea.NewProgram(logModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		reader.Close()
		return fmt.Errorf("log viewer error: %w", err)
	}

	return nil
}

func printError(context string, err error) {
	msg := fmt.Sprintf("\n⚠ %s: %v", context, err)
	fmt.Println(errorStyle.Render(msg))
	fmt.Println(errorStyle.Render("  Check debug.log for more details."))
	logger.Errorf("%s: %v", context, err)
}

func printWarning(context string, err error) {
	msg := fmt.Sprintf("\n⚠ %s: %v", context, err)
	fmt.Println(warnStyle.Render(msg))
	logger.Errorf("%s: %v", context, err)
}

func showAbout() {
	aboutModel := ui.NewAboutModel()
	p := tea.NewProgram(aboutModel)
	p.Run()
}
