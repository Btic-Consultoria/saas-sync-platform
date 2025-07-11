package sage

import (
	"context"
	"database/sql"
	"fmt"
	"saas-sync-platform/internal/shared"
	"time"
)

// Connector handles connections to Sage 200c database.
type Connector struct {
	db     *sql.DB
	config *shared.DatabaseConfig
}

// NewConnector creates a new Sage database connector.
func NewConnector(config *shared.DatabaseConfig) *Connector {
	return &Connector{
		config: config,
	}
}

// Connect establishes connection to Sage database.
func (c *Connector) Connect() error {
	connStr := c.config.GetSageConnectionString()

	var err error
	c.db, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		c.db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Cl.ose closes the database connection
func (c *Connector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// GetCustomers retrieves customers from Sage database
func (c *Connector) GetCustomers(lastSync time.Time) ([]shared.Customer, error) {
	query := `
		SELECT 
            CustomerAccountNumber,
            CustomerName,
            TelephoneNumber,
            EmailAddress,
            MainAddress.Address1,
            MainAddress.City,
            MainAddress.PostCode,
            MainAddress.Country,
            DateTimeModified
        FROM SLCustomers c
        LEFT JOIN PLPostalAddresses MainAddress ON c.MainAddressID = MainAddress.PostalAddressID
        WHERE c.DateTimeModified > ?
        ORDER BY c.DateTimeModified
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
		var phone, email, address, city, postalCode, country sql.NullString

		err := rows.Scan(
			&customer.Code,
			&customer.Name,
			&phone,
			&email,
			&address,
			&city,
			&postalCode,
			&country,
			&customer.ModifiedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}

		// Handle nullable fields.
		customer.Phone = phone.String
		customer.Email = email.String
		customer.Address = address.String
		customer.City = city.String
		customer.PostalCode = postalCode.String
		customer.Country = country.String
		customer.ID = customer.Code // Customer code as ID

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	return customers, nil
}

// GetInvoices retrieves invoices from Sage database
func (c *Connector) GetInvoices(lastSync time.Time) ([]shared.Invoice, error) {
	query := `
        SELECT 
            i.InvoiceNumber,
            i.CustomerAccountNumber,
            i.DocumentSubTotal,
            i.DocumentTaxValue,
            i.DocumentTotalValue,
            i.DocumentDate,
            i.DueDate,
            i.InvoiceStatusID,
            i.DateTimeModified
        FROM SLInvoices i
        WHERE i.DateTimeModified > ?
        ORDER BY i.DateTimeModified
    `

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query, lastSync)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	var invoices []shared.Invoice
	for rows.Next() {
		var invoice shared.Invoice
		var statusID int

		err := rows.Scan(
			&invoice.Number,
			&invoice.CustomerID,
			&invoice.Amount,
			&invoice.TaxAmount,
			&invoice.TotalAmount,
			&invoice.Date,
			&invoice.DueDate,
			&statusID,
			&invoice.ModifiedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice row: %w", err)
		}

		invoice.ID = invoice.Number
		invoice.Status = c.getInvoiceStatus(statusID)

		invoices = append(invoices, invoice)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invoice rows: %w", err)
	}

	return invoices, nil
}

// GetProducts retrieves products from Sage database
func (c *Connector) GetProducts(lastSync time.Time) ([]shared.Product, error) {
	query := `
        SELECT 
            ProductCode,
            ProductName,
            ProductDescription,
            UnitSellingPrice,
            ProductGroupName,
            DateTimeModified
        FROM StockItems s
        LEFT JOIN StockItemGroups g ON s.StockItemGroupID = g.StockItemGroupID
        WHERE s.DateTimeModified > ?
        ORDER BY s.DateTimeModified
    `

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query, lastSync)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []shared.Product
	for rows.Next() {
		var product shared.Product
		var description, category sql.NullString

		err := rows.Scan(
			&product.Code,
			&product.Name,
			&description,
			&product.Price,
			&category,
			&product.ModifiedDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}

		product.ID = product.Code
		product.Description = description.String
		product.Category = category.String

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// TestConnection performs a simple test query to verify database connectivity.
func (c *Connector) TestConnection() error {
	query := "SELECT TOP 1 CustomerAccountNumber FROM SLCustomers"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var customerCode string
	err := c.db.QueryRowContext(ctx, query).Scan(&customerCode)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("test query failed: %w", err)
	}

	return nil
}

// GetLastSyncTime retrieves the last successful sync time for a given sync type.
func (c *Connector) GetLastSyncTime(syncType string) (time.Time, error) {
	// This would typically be stored in a sync log table in the database
	// or retrieved from the SaaS platform

	// For now, return a default time (e. g. last 24 hours).
	return time.Now().AddDate(0, 0, -1), nil
}

// getInvoiceStatus converts Sage invoice status ID to readable status
func (c *Connector) getInvoiceStatus(statusID int) string {
	switch statusID {
	case 1:
		return "Draft"
	case 2:
		return "Pending"
	case 3:
		return "Sent"
	case 4:
		return "Paid"
	case 5:
		return "Overdue"
	case 6:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// ExecuteCustomQuery allows executing custom SQL queries for specific sync requirements.
func (c *Connector) ExecuteCustomQuery(query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return c.db.QueryContext(ctx, query, args...)
}
