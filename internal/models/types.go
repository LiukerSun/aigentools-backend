package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSON is a custom type for handling JSON/JSONB fields in GORM
type JSON map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	if len(bytes) == 0 {
		*j = make(JSON)
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	*j = JSON(result)
	return nil
}
