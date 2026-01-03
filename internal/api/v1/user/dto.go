package user

// UserResponse defines the response structure for user information.
type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Token    string `json:"token,omitempty"`
}
