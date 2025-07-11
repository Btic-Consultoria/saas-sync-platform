package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// ConfigLoader handles loading configuration from various sources.
type ConfigLoader struct {
	configPath string
	envPath    string
}

// NewConfigLoader creates a new configuration loader.
func NewConfigLoader(configPath, envPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
		envPath:    envPath,
	}
}

// LoadConfig loads configuration from JSON file and environment variables.
func (cl *ConfigLoader) LoadConfig() (*AgentConfig, error) {
	config := &AgentConfig{}

	// Load from environment file if it exists.
	if cl.envPath != "" && fileExists(cl.envPath) {
		if err := godotenv.Load(cl.envPath); err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Load from JSON file if it exists.
	if cl.configPath != "" && fileExists(cl.configPath) {
		if err := cl.loadFromJSON(config); err != nil {
			return nil, fmt.Errorf("error loading JSON config: %w", err)
		}
	}

	// Override/supplement with environment variables.
	cl.loadFromEnv(config)

	// Set defaults.
	cl.setDefaults(config)

	return config, nil
}

// loadFromJSON loads configuration from JSON file.
func (cl *ConfigLoader) loadFromJSON(config *AgentConfig) error {
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, config)
}

// loadFromEnv loads configuration from environment variables.
func (cl *ConfigLoader) loadFromEnv(config *AgentConfig) {
	// Database configuration
	if config.Database.Host == "" {
		config.Database.Host = getEnv("SAGE_DB_HOST", "")
	}
	if config.Database.Port == "" {
		config.Database.Port = getEnv("SAGE_DB_PORT", "1433")
	}
	if config.Database.Database == "" {
		config.Database.Database = getEnv("SAGE_DB_NAME", "")
	}
	if config.Database.Username == "" {
		config.Database.Username = getEnv("SAGE_DB_USER", "")
	}
	if config.Database.Password == "" {
		config.Database.Password = getEnv("SAGE_DB_PASSWORD", "")
	}
	if config.Database.LicenseID == "" {
		config.Database.LicenseID = getEnv("LICENSE_ID", "")
	}

	// Bitrix24 configuration.
	if config.Bitrix24 == nil {
		bitrixEndpoint := getEnv("BITRIX_ENDPOINT", "")
		if bitrixEndpoint != "" {
			config.Bitrix24 = &Bitrix24Config{
				APITenant:   bitrixEndpoint,
				PackEmpresa: getBoolEnv("PACK_EMPRESA", false),
			}
		}
	}

	// Company mapping from environment.
	if len(config.Companies) == 0 {
		empresaBitrix := getEnv("EMPRESA_BITRIX", "")
		empresaSage := getEnv("EMPRESA_SAGE", "")
		if empresaBitrix != "" && empresaSage != "" {
			config.Companies = []CompanyMapping{
				{
					BitrixCompany: empresaBitrix,
					SageCompany:   empresaSage,
				},
			}
		}
	}

	// Client code.
	if config.ClientCode == "" {
		config.ClientCode = getEnv("BITRIX_CLIENT_CODE", "")
	}

	// Sync settings
	if config.SyncSettings.IntervalMinutes == 0 {
		config.SyncSettings.IntervalMinutes = getIntEnv("SYNC_INTERVAL_MINUTES", 5)
	}
	if config.SyncSettings.LogLevel == "" {
		config.SyncSettings.LogLevel = getEnv("LOG_LEVEL", "info")
	}
}

// setDefaults sets default values for missing configuration.
func (cl *ConfigLoader) setDefaults(config *AgentConfig) {
	if config.SyncSettings.IntervalMinutes == 0 {
		config.SyncSettings.IntervalMinutes = 5
	}
	if config.SyncSettings.LogLevel == "" {
		config.SyncSettings.LogLevel = "info"
	}
	if config.SaaSConfig.BaseURL == "" {
		config.SaaSConfig.BaseURL = "https://api.btic.cat"
	}
	if !config.SaaSConfig.TLSEnabled {
		config.SaaSConfig.TLSEnabled = true
	}
}

// SaveConfig saves the current configuration to JSON file.
func (cl *ConfigLoader) SaveConfig(config *AgentConfig) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	// Ensure directory exists.
	dir := filepath.Dir(cl.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cl.configPath, data, 0644)
}

// GetDefaultConfigPaths returns default configuration file paths.
func GetDefaultConfigPaths() (configPath, envPath string) {
	homeDir, _ := os.UserHomeDir()
	appDataDir := filepath.Join(homeDir, "AppData", "Roaming", "SageSync")

	configPath = filepath.Join(appDataDir, "config.json")
	envPath = filepath.Join(appDataDir, "config.env")

	return configPath, envPath
}

// GetDevelopmentConfigPaths returns development configuration paths.
func GetDevelopmentConfigPaths() (configPath, envPath string) {
	configPath = "./configs/config.json"
	envPath = "./.env"

	return configPath, envPath
}

// Utility functions
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// ValidateConfig validates the configuration.
func ValidateConfig(config *AgentConfig) error {
	if config.ClientCode == "" {
		return fmt.Errorf("client code is required")
	}

	if !config.Database.IsValid() {
		return fmt.Errorf("database configuration is incomplete")
	}

	if config.Bitrix24 == nil && config.Tickelia == nil {
		return fmt.Errorf("at least one integration (Bitrix24 or Tickelia) must be configured")
	}

	if len(config.Companies) == 0 {
		return fmt.Errorf("at least one company mapping is required")
	}

	return nil
}
