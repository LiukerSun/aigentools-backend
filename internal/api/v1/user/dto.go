package user

import "time"

// UserResponse defines the response structure for user information.
type UserResponse struct {
	ID            uint       `json:"id"`
	Username      string     `json:"username"`
	Role          string     `json:"role"`
	IsActive      bool       `json:"is_active"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
	Token         string     `json:"token,omitempty"`
}
