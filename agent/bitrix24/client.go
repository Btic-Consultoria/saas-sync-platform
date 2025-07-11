// agent/bitrix24/client.go
package bitrix24

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"saas-sync-platform/internal/shared"
)

// Client handles communication with Bitrix24 API
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     *shared.Bitrix24Config
}

// NewClient creates a new Bitrix24 API client
func NewClient(config *shared.Bitrix24Config) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(config.APITenant, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		config:     config,
	}
}

// Contact represents a Bitrix24 contact
type Contact struct {
	ID       string            `json:"ID,omitempty"`
	Name     string            `json:"NAME"`
	LastName string            `json:"LAST_NAME,omitempty"`
	Phone    []PhoneField      `json:"PHONE,omitempty"`
	Email    []EmailField      `json:"EMAIL,omitempty"`
	Comments string            `json:"COMMENTS,omitempty"`
	Fields   map[string]string `json:"-"` // Additional custom fields
}

// PhoneField represents a phone number in Bitrix24
type PhoneField struct {
	Value     string `json:"VALUE"`
	ValueType string `json:"VALUE_TYPE"`
	TypeID    string `json:"TYPE_ID"`
}

// EmailField represents an email in Bitrix24
type EmailField struct {
	Value     string `json:"VALUE"`
	ValueType string `json:"VALUE_TYPE"`
	TypeID    string `json:"TYPE_ID"`
}

// APIResponse represents a standard Bitrix24 API response
type APIResponse struct {
	Result interface{} `json:"result"`
	Error  *APIError   `json:"error,omitempty"`
	Time   struct {
		Start      float64 `json:"start"`
		Finish     float64 `json:"finish"`
		Duration   float64 `json:"duration"`
		Processing float64 `json:"processing"`
		DateStart  string  `json:"date_start"`
		DateFinish string  `json:"date_finish"`
	} `json:"time"`
}

// APIError represents a Bitrix24 API error
type APIError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Bitrix24 API error %s: %s", e.ErrorCode, e.ErrorDescription)
}

// CreateContact creates a new contact in Bitrix24
func (c *Client) CreateContact(customer *shared.Customer) (*Contact, error) {
	contactFields := c.customerToContact(customer)

	data := map[string]interface{}{
		"fields": contactFields,
	}

	var response APIResponse
	err := c.makeRequest("crm.contact.add", data, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	if response.Error != nil {
		return nil, response.Error
	}

	// Create a Contact struct to return
	contact := &Contact{
		Name:     customer.Name,
		Comments: fmt.Sprintf("Synced from Sage 200c - Customer Code: %s", customer.Code),
	}

	// Set phone if available
	if customer.Phone != "" {
		contact.Phone = []PhoneField{
			{
				Value:     customer.Phone,
				ValueType: "WORK",
				TypeID:    "PHONE",
			},
		}
	}

	// Set email if available
	if customer.Email != "" {
		contact.Email = []EmailField{
			{
				Value:     customer.Email,
				ValueType: "WORK",
				TypeID:    "EMAIL",
			},
		}
	}

	// The result should contain the new contact ID
	if contactID, ok := response.Result.(float64); ok {
		contact.ID = fmt.Sprintf("%.0f", contactID)
		log.Printf("Created Bitrix24 contact: %s (ID: %s)", customer.Name, contact.ID)
	}

	return contact, nil
}

// UpdateContact updates an existing contact in Bitrix24
func (c *Client) UpdateContact(contactID string, customer *shared.Customer) error {
	contact := c.customerToContact(customer)

	data := map[string]interface{}{
		"id":     contactID,
		"fields": contact,
	}

	var response APIResponse
	err := c.makeRequest("crm.contact.update", data, &response)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	if response.Error != nil {
		return response.Error
	}

	log.Printf("Updated Bitrix24 contact: %s (ID: %s)", customer.Name, contactID)
	return nil
}

// FindContactByName searches for a contact by name
func (c *Client) FindContactByName(name string) (*Contact, error) {
	data := map[string]interface{}{
		"filter": map[string]string{
			"NAME": name,
		},
		"select": []string{"ID", "NAME", "LAST_NAME", "PHONE", "EMAIL"},
	}

	var response APIResponse
	err := c.makeRequest("crm.contact.list", data, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to search contact: %w", err)
	}

	if response.Error != nil {
		return nil, response.Error
	}

	// Parse the results
	if resultArray, ok := response.Result.([]interface{}); ok && len(resultArray) > 0 {
		if contactData, ok := resultArray[0].(map[string]interface{}); ok {
			contact := &Contact{}

			// Handle ID as either string or number
			if id, ok := contactData["ID"].(string); ok {
				contact.ID = id
			} else if id, ok := contactData["ID"].(float64); ok {
				contact.ID = fmt.Sprintf("%.0f", id)
			}

			if name, ok := contactData["NAME"].(string); ok {
				contact.Name = name
			}
			if lastName, ok := contactData["LAST_NAME"].(string); ok {
				contact.LastName = lastName
			}
			return contact, nil
		}
	}

	return nil, fmt.Errorf("contact not found")
}

// SyncCustomer syncs a Sage customer to Bitrix24 (create or update)
func (c *Client) SyncCustomer(customer *shared.Customer) error {
	// Try to find existing contact
	existingContact, err := c.FindContactByName(customer.Name)
	if err == nil && existingContact != nil {
		// Contact exists, update it
		return c.UpdateContact(existingContact.ID, customer)
	}

	// Contact doesn't exist, create new one
	_, err = c.CreateContact(customer)
	return err
}

// SyncCustomers syncs multiple customers to Bitrix24
func (c *Client) SyncCustomers(customers []shared.Customer) error {
	successCount := 0
	errorCount := 0

	log.Printf("Starting sync of %d customers to Bitrix24", len(customers))

	for _, customer := range customers {
		if err := c.SyncCustomer(&customer); err != nil {
			log.Printf("Failed to sync customer %s: %v", customer.Name, err)
			errorCount++
		} else {
			successCount++
		}

		// Rate limiting - pause between requests
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Bitrix24 sync completed: %d successful, %d errors", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("sync completed with %d errors", errorCount)
	}

	return nil
}

// TestConnection tests the Bitrix24 API connection
func (c *Client) TestConnection() error {
	data := map[string]interface{}{}

	var response APIResponse
	err := c.makeRequest("crm.contact.fields", data, &response)
	if err != nil {
		return fmt.Errorf("Bitrix24 connection test failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("Bitrix24 API error: %v", response.Error)
	}

	log.Println("Bitrix24 connection test successful")
	return nil
}

// customerToContact converts a Sage customer to Bitrix24 contact format
func (c *Client) customerToContact(customer *shared.Customer) map[string]interface{} {
	contact := map[string]interface{}{
		"NAME":     customer.Name,
		"COMMENTS": fmt.Sprintf("Synced from Sage 200c - Customer Code: %s", customer.Code),
	}

	// Add phone if available
	if customer.Phone != "" {
		contact["PHONE"] = []PhoneField{
			{
				Value:     customer.Phone,
				ValueType: "WORK",
				TypeID:    "PHONE",
			},
		}
	}

	// Add email if available
	if customer.Email != "" {
		contact["EMAIL"] = []EmailField{
			{
				Value:     customer.Email,
				ValueType: "WORK",
				TypeID:    "EMAIL",
			},
		}
	}

	// Add address fields if available
	if customer.Address != "" {
		contact["ADDRESS"] = customer.Address
	}
	if customer.City != "" {
		contact["ADDRESS_CITY"] = customer.City
	}
	if customer.PostalCode != "" {
		contact["ADDRESS_POSTAL_CODE"] = customer.PostalCode
	}
	if customer.Country != "" {
		contact["ADDRESS_COUNTRY"] = customer.Country
	}

	return contact
}

// makeRequest makes a request to the Bitrix24 API
func (c *Client) makeRequest(method string, data map[string]interface{}, result interface{}) error {
	// Prepare the request URL
	requestURL := fmt.Sprintf("%s/%s", c.baseURL, method)

	// Prepare the request body
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}
