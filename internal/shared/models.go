package shared

import (
	"fmt"
	"time"
)

// AgentConfig represents the complete configuration for the sync agent.
type AgentConfig struct {
	ClientCode   string           `json:"CodigoCliente" mapstructure:"client_code"`
	Database     DatabaseConfig   `json:"DB" mapstructure:"database"`
	Bitrix24     *Bitrix24Config  `json:"Bitrix24,omitempty" mapstructure:"bitrix24"`
	Tickelia     *TickeliaConfig  `json:"Tickelia,omitempty" mapstructure:"tickelia"`
	Companies    []CompanyMapping `json:"Empresas" mapstructure:"companies"`
	SyncSettings SyncSettings     `json:"SyncSettings,omitempty" mapstructure:"saas"`
	SaaSConfig   SaaSConnection    `json:"SaaS,omitempty" mapstructure:"saas"`
}

// DatabaseConfig contains Sage 200c database connection details.
type DatabaseConfig struct {
	Host      string `json:"DB_Host" mapstructure:"host"`
	HostSage  string `json:"DB_Host_Sage" mapstructure:"host_sage"`
	Port      string `json:"DB_Port" mapstructure:"port"`
	Database  string `json:"DB_Database" mapstructure:"database"`
	Username  string `json:"DB_Username" mapstructure:"username"`
	Password  string `json:"DB_Password" mapstructure:"password"`
	LicenseID string `json:"IdLlicencia" mapstructure:"license_id"`
}

// Bitrix24Config contains Bitrix24 integration settings.
type Bitrix24Config struct {
	APITenant   string `json:"API_Tenant" mapstructure:"api_tenant"`
	PackEmpresa bool   `json:"pack_empresa" mapstructure:"pack_empresa"`
}

// TickeliaConfig contains Tickelia integration settings.
type TickeliaConfig struct {
	APIEndpoint string `json:"API_Endpoint" mapstructure:"api_endpoint"`
	APIKey      string `json:"API_Key" mapstructure:"api_key"`
	Environment string `json:"Environment" mapstructure:"environment"` // "dev", "prod"
}

// CompanyMapping maps Sage companies to external service companies.
type CompanyMapping struct {
	BitrixCompany string `json:"EmpresaBitrix" mapstructure:"bitrix_company"`
	SageCompany   string `json:"EmpresaSage" mapstructure:"sage_company"`
}

// SyncSettings contains synchronization preferences.
type SyncSettings struct {
	IntervalMinutes int      `json:"interval_minutes" mapstructure:"interval_minutes"`
	EnabledModules  []string `json:"enabled_modules" mapstructure:"enabled_modules"`
	LogLevel        string   `json:"log_level" mapstructure:"log_level"`
}

// SaaSConnection contains connection details for the SaaS platform.
type SaaSConnection struct {
	BaseURL    string `json:"base_url" mapstructure:"base_url"`
	APIKey     string `json:"api_key" mapstructure:"api_key"`
	ClientID   string `json:"client_id" mapstructure:"client_id"`
	TLSEnabled bool   `json:"tls_enabled" mapstructure:"tls_enabled"`
}

// GetSageConnectionsTring returns the SQL Server connection string.
func (db *DatabaseConfig) GetSageConnectionString() string {
	// Handle SQL Server named instance format.
	host := db.Host
	if host == "" {
		host = db.HostSage
	}

	return fmt.Sprintf("server=%s;port=%s;database=%s;user id=%s;password=%s;encrypt=false",
		host, db.Port, db.Database, db.Username, db.Password)
}

// IsValid checks if the database configuration is complete.
func (db *DatabaseConfig) IsValid() bool {
	return db.Host != "" && db.Database != "" && db.Username != "" && db.Password != ""
}

// GetSaaSURL returns the complete SaaS API URL.
func (saas *SaaSConnection) GetSaaSURL() string {
	if saas.BaseURL == "" {
		return "https://api.btic.cat"
	}
	return saas.BaseURL
}

// SyncTask represents a synch task from the SaaS platform.
type SyncTask struct {
	ID          string                 `json:"id"`
	ClientID    string                 `json:"client_id"`
	Type        string                 `json:"type"`        // "customers", "invoices", "products"
	Integration string                 `json:"integration"` // "bitrix24", "tickelia"
	Config      map[string]interface{} `json:"config"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	LastSync    *time.Time             `json:"last_sync,omitempty"`
}

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	TaskID       string    `json:"task_id"`
	ClientID     string    `json:"client_id"`
	Success      bool      `json:"success"`
	RecordsCount int       `json:"records_count"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

// Customer represents a Sage customer record.
type Customer struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	City         string    `json:"city"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
	ModifiedDate time.Time `json:"modified_date"`
}

// Invoice represents a Sage invoice record.
type Invoice struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	CustomerID   string    `json:"customer_id"`
	Amount       float64   `json:"amount"`
	TaxAmount    float64   `json:"tax_amount"`
	TotalAmount  float64   `json:"total_amount"`
	Date         time.Time `json:"date"`
	DueDate      time.Time `json:"due_date"`
	Status       string    `json:"status"`
	ModifiedDate time.Time `json:"modified_date"`
}

// Product represents a Sage product record
type Product struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Category     string    `json:"category"`
	ModifiedDate time.Time `json:"modified_date"`
}
