package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"saas-sync-platform/internal/shared"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/getlantern/systray"
)

type TrayAgent struct {
	config       *shared.AgentConfig
	configLoader *shared.ConfigLoader
	sageDB       *sql.DB
	isRunning    bool
	lastSync     time.Time

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
	// Set the icon and tooltip
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

	// Connect to Sage database
	if err := a.connectToSage(); err != nil {
		a.showError("Failed to connect to Sage database: " + err.Error())
		a.updateStatus("Connection failed - Check configuration")
		return
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

	// Close database connection
	if a.sageDB != nil {
		a.sageDB.Close()
		a.sageDB = nil
	}

	log.Println("Sync stopped successfully")
}

func (a *TrayAgent) connectToSage() error {
	connStr := a.config.Database.GetSageConnectionString()

	log.Printf("Connecting to Sage database: %s:%s/%s",
		a.config.Database.Host, a.config.Database.Port, a.config.Database.Database)

	var err error
	a.sageDB, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.sageDB.PingContext(ctx); err != nil {
		a.sageDB.Close()
		a.sageDB = nil
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to Sage database")
	return nil
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

func (a *TrayAgent) performSync() {
	a.updateStatus("Syncing...")
	a.lastSync = time.Now()

	log.Println("Starting sync operation...")

	// Test database connection
	if err := a.testSageConnection(); err != nil {
		a.showError("Sync failed: " + err.Error())
		a.updateStatus("Sync failed - Will retry")
		return
	}

	// TODO: Add actual sync logic here:
	// 1. Fetch sync tasks from SaaS platform
	// 2. Read data from Sage database
	// 3. Send data to external services (Bitrix24, etc.)
	// 4. Update sync status

	log.Println("Sync completed successfully")

	status := fmt.Sprintf("Last sync: %s", a.lastSync.Format("15:04:05"))
	a.updateStatus(status)
}

func (a *TrayAgent) testSageConnection() error {
	// Simple test query to verify connection
	query := `SELECT TOP 1 CustomerAccountNumber, CustomerName FROM SLCustomers`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var customerCode, customerName string
	err := a.sageDB.QueryRowContext(ctx, query).Scan(&customerCode, &customerName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("database test query failed: %w", err)
	}

	if err != sql.ErrNoRows {
		log.Printf("Database test successful - found customer: %s (%s)", customerCode, customerName)
	} else {
		log.Println("Database test successful - no customers found but connection works")
	}

	return nil
}

func (a *TrayAgent) updateStatus(status string) {
	a.mStatus.SetTitle("Status: " + status)
	systray.SetTooltip("Sage Sync Agent - " + status)
}

func (a *TrayAgent) showError(message string) {
	log.Printf("ERROR: %s", message)
	// TODO: Show Windows notification or dialog
}

func (a *TrayAgent) showConfiguration() {
	log.Println("=== Current Configuration ===")
	log.Printf("Client Code: %s", a.config.ClientCode)
	log.Printf("Database: %s:%s/%s", a.config.Database.Host, a.config.Database.Port, a.config.Database.Database)
	log.Printf("Sync Interval: %d minutes", a.config.SyncSettings.IntervalMinutes)

	if a.config.Bitrix24 != nil {
		log.Printf("Bitrix24: %s", a.config.Bitrix24.APITenant)
	}

	log.Printf("Companies: %d configured", len(a.config.Companies))
	for i, company := range a.config.Companies {
		log.Printf("  %d. Bitrix: %s -> Sage: %s", i+1, company.BitrixCompany, company.SageCompany)
	}

	// TODO: Open configuration dialog or web interface
}

func (a *TrayAgent) openLogs() {
	log.Println("Opening logs...")
	// TODO: Open log file in default text editor
	// exec.Command("notepad", "path/to/logfile.log").Start()
}
