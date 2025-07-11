// cmd/tray-agent/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"saas-sync-platform/agent/bitrix24"
	"saas-sync-platform/agent/sage"
	"saas-sync-platform/internal/shared"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/getlantern/systray"
)

type TrayAgent struct {
	config         *shared.AgentConfig
	configLoader   *shared.ConfigLoader
	sageConnector  *sage.Connector
	bitrix24Client *bitrix24.Client
	isRunning      bool
	lastSync       time.Time

	// Menu items
	mStatus *systray.MenuItem
	mStart  *systray.MenuItem
	mStop   *systray.MenuItem
	mConfig *systray.MenuItem
	mLogs   *systray.MenuItem
	mQuit   *systray.MenuItem
}

func main() {
	log.Println("Starting Sage Sync Agent...")

	// Run the system tray application
	systray.Run(onReady, onExit)
}

func onReady() {
	agent := &TrayAgent{}

	log.Println("Initializing tray agent...")

	// Initialize system tray UI
	agent.initSystemTray()

	// Load configuration
	if err := agent.loadConfiguration(); err != nil {
		agent.showError("Failed to load configuration: " + err.Error())
		agent.updateStatus("Configuration Error")
		return
	}

	// Agent is ready
	agent.updateStatus("Ready - Click 'Start Sync' to begin")
	log.Printf("Sage Sync Agent ready for client: %s", agent.config.ClientCode)
}

func onExit() {
	log.Println("Sage Sync Agent shutting down...")
}

func (a *TrayAgent) initSystemTray() {
	// Set the title and tooltip (no icon for now)
	systray.SetTitle("Sage Sync")
	systray.SetTooltip("Sage 200c Synchronization Agent")

	// Create menu items
	a.mStatus = systray.AddMenuItem("Status: Initializing...", "Current sync status")
	a.mStatus.Disable()

	systray.AddSeparator()

	a.mStart = systray.AddMenuItem("🔄 Start Sync", "Start synchronization")
	a.mStop = systray.AddMenuItem("⏹️ Stop Sync", "Stop synchronization")
	a.mStop.Disable()

	systray.AddSeparator()

	a.mConfig = systray.AddMenuItem("⚙️ Configuration", "View configuration details")
	a.mLogs = systray.AddMenuItem("📋 View Logs", "Open log file")

	systray.AddSeparator()

	a.mQuit = systray.AddMenuItem("❌ Exit", "Exit the application")

	// Handle menu clicks
	go a.handleMenuClicks()
}

func (a *TrayAgent) handleMenuClicks() {
	for {
		select {
		case <-a.mStart.ClickedCh:
			a.startSync()

		case <-a.mStop.ClickedCh:
			a.stopSync()

		case <-a.mConfig.ClickedCh:
			a.showConfiguration()

		case <-a.mLogs.ClickedCh:
			a.openLogs()

		case <-a.mQuit.ClickedCh:
			if a.isRunning {
				a.stopSync()
			}
			systray.Quit()
			return
		}
	}
}

func (a *TrayAgent) loadConfiguration() error {
	// Determine if we're in development mode
	isDev := os.Getenv("ENV") == "development"

	var configPath, envPath string
	if isDev {
		configPath, envPath = shared.GetDevelopmentConfigPaths()
		log.Println("Development mode: using local config files")
	} else {
		configPath, envPath = shared.GetDefaultConfigPaths()
		log.Println("Production mode: using user config files")
	}

	// Create config loader
	a.configLoader = shared.NewConfigLoader(configPath, envPath)

	// Load configuration
	config, err := a.configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.config = config

	// Validate configuration
	if err := shared.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	log.Printf("Configuration loaded successfully for client: %s", config.ClientCode)
	return nil
}

func (a *TrayAgent) startSync() {
	if a.isRunning {
		return
	}

	log.Println("Starting sync operation...")
	a.updateStatus("Starting...")

	// Initialize Sage connector
	a.sageConnector = sage.NewConnector(&a.config.Database)
	if err := a.sageConnector.Connect(); err != nil {
		a.showError("Failed to connect to Sage database: " + err.Error())
		a.updateStatus("Sage connection failed")
		return
	}

	// Initialize Bitrix24 client
	if a.config.Bitrix24 != nil {
		a.bitrix24Client = bitrix24.NewClient(a.config.Bitrix24)
		if err := a.bitrix24Client.TestConnection(); err != nil {
			a.showError("Failed to connect to Bitrix24: " + err.Error())
			a.updateStatus("Bitrix24 connection failed")
			a.sageConnector.Close()
			return
		}
	}

	a.isRunning = true
	a.mStart.Disable()
	a.mStop.Enable()

	// Update status
	interval := a.config.SyncSettings.IntervalMinutes
	a.updateStatus(fmt.Sprintf("Connected - Syncing every %d minutes", interval))

	log.Printf("Sync started with %d minute interval", interval)

	// Start sync loop
	go a.syncLoop()
}

func (a *TrayAgent) stopSync() {
	if !a.isRunning {
		return
	}

	log.Println("Stopping sync operation...")

	a.isRunning = false
	a.mStart.Enable()
	a.mStop.Disable()
	a.updateStatus("Stopped")

	// Close connections
	if a.sageConnector != nil {
		a.sageConnector.Close()
		a.sageConnector = nil
	}

	a.bitrix24Client = nil

	log.Println("Sync stopped successfully")
}

func (a *TrayAgent) openLogs() {
	log.Println("Opening logs...")
	// TODO: Open log file in default text editor
	// exec.Command("notepad", "path/to/logfile.log").Start()
}

func (a *TrayAgent) updateStatus(status string) {
	a.mStatus.SetTitle("Status: " + status)
	systray.SetTooltip("Sage Sync Agent - " + status)
}

func (a *TrayAgent) showError(message string) {
	log.Printf("ERROR: %s", message)
	// TODO: Show Windows notification or dialog in the future
}

func (a *TrayAgent) showConfiguration() {
	log.Println("=== Current Configuration ===")
	log.Printf("Client Code: %s", a.config.ClientCode)
	log.Printf("Database: %s:%s/%s", a.config.Database.Host, a.config.Database.Port, a.config.Database.Database)
	log.Printf("Sync Interval: %d minutes", a.config.SyncSettings.IntervalMinutes)

	if a.config.Bitrix24 != nil {
		log.Printf("Bitrix24: %s", a.config.Bitrix24.APITenant)
		log.Printf("Pack Empresa: %t", a.config.Bitrix24.PackEmpresa)
	} else {
		log.Println("Bitrix24: Not configured")
	}

	log.Printf("Companies: %d configured", len(a.config.Companies))
	for i, company := range a.config.Companies {
		log.Printf("  %d. Bitrix: %s -> Sage: %s", i+1, company.BitrixCompany, company.SageCompany)
	}

	// Show database info if connected
	if a.sageConnector != nil {
		if info, err := a.sageConnector.GetDatabaseInfo(); err == nil {
			log.Printf("Database Info:")
			for key, value := range info {
				log.Printf("  %s: %v", key, value)
			}
		}
	}
}

func (a *TrayAgent) performSync() {
	a.updateStatus("Syncing...")
	a.lastSync = time.Now()

	log.Println("Starting sync operation...")

	// Test Sage connection first
	if err := a.sageConnector.TestConnection(); err != nil {
		a.showError("Sage connection test failed: " + err.Error())
		a.updateStatus("Sync failed - Sage connection")
		return
	}

	// Get recent customers from Sage (last 24 hours if no previous sync)
	since := a.lastSync.Add(-24 * time.Hour)
	if !a.lastSync.IsZero() {
		since = a.lastSync.Add(-time.Duration(a.config.SyncSettings.IntervalMinutes) * time.Minute)
	}

	customers, err := a.sageConnector.GetRecentCustomers(since)
	if err != nil {
		a.showError("Failed to read Sage customers: " + err.Error())
		a.updateStatus("Sync failed - Sage data")
		return
	}

	log.Printf("Found %d customers to sync", len(customers))

	// Sync to Bitrix24 if configured
	if a.bitrix24Client != nil {
		a.updateStatus(fmt.Sprintf("Syncing %d customers to Bitrix24...", len(customers)))

		if err := a.bitrix24Client.SyncCustomers(customers); err != nil {
			a.showError("Bitrix24 sync failed: " + err.Error())
			a.updateStatus("Sync failed - Bitrix24")
			return
		}

		log.Printf("Successfully synced %d customers to Bitrix24", len(customers))
	}

	// Update status with results
	status := fmt.Sprintf("Last sync: %s (%d customers)",
		a.lastSync.Format("15:04:05"), len(customers))
	a.updateStatus(status)

	log.Printf("Sync completed successfully: %d customers processed", len(customers))
}

func (a *TrayAgent) syncLoop() {
	interval := time.Duration(a.config.SyncSettings.IntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Perform initial sync
	a.performSync()

	// Continue syncing until stopped
	for a.isRunning {
		select {
		case <-ticker.C:
			if a.isRunning {
				a.performSync()
			}
		case <-time.After(1 * time.Second):
			// Check if we should stop (allows quick exit)
			if !a.isRunning {
				return
			}
		}
	}
}
