package payment

// Driver is the interface that all payment drivers must implement
type Driver interface {
	// SetConfig sets the configuration for the driver
	SetConfig(config map[string]interface{}) error

	// Pay initiates a payment and returns the jump URL
	// notifyURL: The base notify URL, driver should append necessary params if needed (though the user requirement says UUID is in the path)
	// Actually, the Service will construct the notify URL with UUID, so here we just pass the full notify URL.
	Pay(orderID string, amount float64, notifyURL string, returnURL string, params map[string]interface{}) (string, error)

	// Notify verifies the callback parameters
	// Returns: isValid, orderID, externalID, error
	Notify(params map[string]interface{}) (bool, string, string, error)
}
