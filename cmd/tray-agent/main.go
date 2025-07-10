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
	// Run the system tray appliction
	systray.Run(onReady, onExit)
}

func onReady() {
	agent := &TrayAgent{}

	// Initialize system tray UI
	agent.initSystemTray()

	// Load configuration
	if err := agent.loadConfiguration(); err != nil {
		agent.showError("Failed to load configuration: " + err.Error())
		return
	}

	// Start the agent
	agent.run()
}

func onExit() {
	log.Println("Sage Sync Agent shutting down...")
}

func (a *TrayAgent) initSystemTray() {
	// Set the icon and tooltip.
	systray.SetTitle("Sage Sync")
	systray.SetTooltip("Sage 200c Synchronization Agent")

	// Create menu items
	a.mStatus = systray.AddMenuItem("Status: Initializing...", "Current sync status")
	a.mStatus.Disable()

	systray.AddSeparator()

	a.mStart = systray.AddMenuItem("Start Sync", "Start synchronization")
	a.mStop = systray.AddMenuItem("Stop Sync", "Stop synchronization")
	a.mStop.Disable()

	systray.AddSeparator()

	a.mConfig = systray.AddMenuItem("Configuration", "Open configuration")
	a.mLogs = systray.AddMenuItem("View Logs", "Open log file")

	systray.AddSeparator()

	a.mQuit = systray.AddMenuItem("Exit", "Exit the application")

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
			a.openConfiguration()

		case <-a.mLogs.ClickedCh:
			a.openLogs()

		case <-a.mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (a *TrayAgent) LoadConfiguration() error {
	// Check if we're in development mode
	isDev := os.Getenv("ENV") == "development"

	var configPath, envPath string
	if isDev {
		configPath, envPath = shared.GetDefaultConfigPaths()
		log.Println("Development mode: using local config files")
	} else {
		configPath, envPath = shared.GetDefaultConfigPaths()
		log.Println("Production mode: using user config files")
	}

	a.configLoader = shared.NewConfigLoader(configPath, envPath)

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

func (a *TrayAgent) run() {
	a.updateStatus("Ready - Click 'Start Sync' to begin")
	log.Println("Sage Sync Agent is ready")
}

func (a *TrayAgent) startSync() {
	if a.isRunning {
		return
	}

	a.updateStatus("Starting...")

	// Connect to Sage database.
	if err := a.connectToSage(); err != nil {
		a.showError("Failed to connect to Sage database: " + err.Error())
		a.updateStatus("Connection failed - Check configuration")
		return
	}

	a.isRunning = true
	a.mStart.Disable()
	a.mStop.Enable()

	// Update status.
	interval := a.config.SyncSettings.IntervalMinutes
	a.updateStatus(fmt.Sprintf("Connected - Syncing every %d minutes", interval))

	log.Printf("Starting sync with interval: %d minutes", interval)

	// Start sync loop
	go a.syncLoop()
}

func (a *TrayAgent) stopSync() {
	if !a.isRunning {
		return
	}

	a.isRunning = false
	a.mStart.Enable()
	a.mStop.Disable()
	a.updateStatus("Stopped")

	if a.sageDB != nil {
		a.sageDB.Close()
		a.sageDB = nil
	}

	log.Println("Sync stopped")
}

func (a *TrayAgent) connectToSage() error {
	connStr := a.config.Database.GetSageConnectionString()
	log.Printf("Connecting to Sage database: %s", fmt.Sprintf("server=%s;database=%s", a.config.Database.Host, a.config.Database.Database))

	var err error
	a.sageDB, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.sageDB.PingContext(ctx); err != nil {
		a.sageDB.Close()
		a.sageDB = nil
		return fmt.Errorf("failed to ping database: %w", err)
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

	for a.isRunning {
		select {
		case <-ticker.C:
			if a.isRunning {
				a.performSync()
			}
		case <-time.After(1 * time.Second):
			// Check if we should stop
			if !a.isRunning {
				return
			}
		}
	}
}

func (a *TrayAgent) performSync() {
	a.updateStatus("Syncing...")
	a.lastSync = time.Now()

	log.Println("Starting sync operation")

	// For now, let's just test the database connection and read some data
	if err := a.testSageConnection(); err != nil {
		a.showError("Sync failed: " + err.Error())
		a.updateStatus("Sync failed - Will retry")
		return
	}

	// Here we would:
	// 1. Fetch sync tasks from SaaS platform
	// 2. Read data from Sage database
	// 3. Send data to SaaS platform
	// 4. Update sync status

	// For now, just log success.
	log.Println("Sync completed successfully")

	status := fmt.Sprintf("Last sync: %s", a.lastSync.Format("15:04:05"))
	a.updateStatus(status)
}

func (a *TrayAgent) testSageConnection() error {
	// Test query to verify connection and read some data
	query := `
	SELECT TOP 5
		CustomerID,
		CustomerName,
	ModifiedDate,
	FROM SLCustomers
	ORDER BY ModifiedDate DESC
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := a.sageDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var customerId, customerName string
		var modifiedDate time.Time

		if err := rows.Scan(&customerId, &customerName, &modifiedDate); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		log.Printf("Customer: %s - %s (Modified: %s)", customerId, customerName, modifiedDate.Format("2006-01-02"))
		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows error: %w", err)
	}

	log.Printf("Successfully read %d customers from Sage", count)
	return nil
}

func (a *TrayAgent) updateStatus(status string) {
	a.mStatus.SetTitle("Status: " + status)
	systray.SetTooltip("Sage Sync Agent - " + status)
}

func (a *TrayAgent) showError(message string) {
	log.Printf("ERROR: %s", message)
	// In a real implementation, we might show a Windows notification
	// or write to a visible log file.
}

func (a *TrayAgent) openConfiguration() {
	log.Println("Opening configuration...")
	// This could:
	// 1. Open the config file in notepad
	// 2. Open a simple config dialog
	// 3. Open your Saas web interface for configuration.
}

func (a *TrayAgent) openLogs() {
	log.Println("Opening logs...")
	// Open the log file in notepad or default text editor.
}
