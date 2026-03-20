package client

import "time"

type AuthInfo struct {
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	SavedAt      time.Time `json:"saved_at"`
}

// IsExpired проверяет, истёк ли токен
func (a *AuthInfo) IsExpired() bool {
	if a == nil {
		return true
	}
	expirationTime := a.SavedAt.Add(time.Duration(a.ExpiresIn) * time.Second)
	return time.Now().After(expirationTime)
}

type RegisterLoginInfo struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
