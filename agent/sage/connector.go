// agent/sage/connector.go
package sage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"saas-sync-platform/internal/shared"
)

// Connector handles connections to Sage 200c database
type Connector struct {
	db     *sql.DB
	config *shared.DatabaseConfig
}

// NewConnector creates a new Sage database connector
func NewConnector(config *shared.DatabaseConfig) *Connector {
	return &Connector{
		config: config,
	}
}

// Connect establishes connection to Sage database
func (c *Connector) Connect() error {
	connStr := c.config.GetSageConnectionString()

	log.Printf("Connecting to Sage database: %s:%s/%s",
		c.config.Host, c.config.Port, c.config.Database)

	var err error
	c.db, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		c.db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to Sage database")
	return nil
}

// Close closes the database connection
func (c *Connector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetRecentCustomers retrieves customers modified since lastSync
func (c *Connector) GetRecentCustomers(lastSync time.Time) ([]shared.Customer, error) {
	query := `
        SELECT TOP 100
            CustomerAccountNumber,
            CustomerName,
            TelephoneNumber,
            FaxNumber,
            EmailAddress,
            DateTimeModified
        FROM SLCustomers 
        WHERE DateTimeModified > ?
        ORDER BY DateTimeModified DESC
    `

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query, lastSync)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers: %w", err)
	}
	defer rows.Close()

	var customers []shared.Customer
	for rows.Next() {
		var customer shared.Customer
		var phone, fax, email sql.NullString

		err := rows.Scan(
			&customer.Code,
			&customer.Name,
			&phone,
			&fax,
			&email,
			&customer.ModifiedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}

		// Set ID and handle nullable fields
		customer.ID = customer.Code
		customer.Phone = phone.String
		customer.Email = email.String

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	log.Printf("Found %d customers modified since %v", len(customers), lastSync)
	return customers, nil
}

// GetCustomerDetails retrieves detailed customer information including addresses
func (c *Connector) GetCustomerDetails(customerCode string) (*shared.Customer, error) {
	query := `
        SELECT 
            c.CustomerAccountNumber,
            c.CustomerName,
            c.TelephoneNumber,
            c.FaxNumber,
            c.EmailAddress,
            c.WebSiteURL,
            addr.Address1,
            addr.Address2,
            addr.City,
            addr.PostCode,
            addr.Country,
            c.DateTimeModified
        FROM SLCustomers c
        LEFT JOIN PLPostalAddresses addr ON c.MainAddressID = addr.PostalAddressID
        WHERE c.CustomerAccountNumber = ?
    `

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var customer shared.Customer
	var phone, fax, email, website, address1, address2, city, postalCode, country sql.NullString

	err := c.db.QueryRowContext(ctx, query, customerCode).Scan(
		&customer.Code,
		&customer.Name,
		&phone,
		&fax,
		&email,
		&website,
		&address1,
		&address2,
		&city,
		&postalCode,
		&country,
		&customer.ModifiedDate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer %s not found", customerCode)
		}
		return nil, fmt.Errorf("failed to query customer details: %w", err)
	}

	// Set fields
	customer.ID = customer.Code
	customer.Phone = phone.String
	customer.Email = email.String
	customer.Address = address1.String
	if address2.String != "" {
		customer.Address += ", " + address2.String
	}
	customer.City = city.String
	customer.PostalCode = postalCode.String
	customer.Country = country.String

	return &customer, nil
}

// TestConnection performs a simple test to verify database connectivity
func (c *Connector) TestConnection() error {
	query := "SELECT TOP 1 CustomerAccountNumber, CustomerName FROM SLCustomers"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var customerCode, customerName string
	err := c.db.QueryRowContext(ctx, query).Scan(&customerCode, &customerName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("test query failed: %w", err)
	}

	if err != sql.ErrNoRows {
		log.Printf("Database test successful - found customer: %s (%s)", customerCode, customerName)
	} else {
		log.Println("Database test successful - no customers found but connection works")
	}

	return nil
}

// GetCustomerCount returns the total number of customers in the database
func (c *Connector) GetCustomerCount() (int, error) {
	query := "SELECT COUNT(*) FROM SLCustomers"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int
	err := c.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count customers: %w", err)
	}

	return count, nil
}

// GetDatabaseInfo returns basic information about the Sage database
func (c *Connector) GetDatabaseInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get customer count
	customerCount, err := c.GetCustomerCount()
	if err != nil {
		return nil, err
	}
	info["customer_count"] = customerCount

	// Get database version info
	query := "SELECT @@VERSION"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var version string
	err = c.db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		log.Printf("Could not get database version: %v", err)
		info["database_version"] = "Unknown"
	} else {
		info["database_version"] = version
	}

	// Get current database name
	info["database_name"] = c.config.Database
	info["host"] = c.config.Host
	info["port"] = c.config.Port

	return info, nil
}
