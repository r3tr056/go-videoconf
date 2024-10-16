package database

type Token struct {
	AccessToken  string `json:"accessToken" example:"Access Token"`
	RefreshToken string `json:"refreshToken" example:"Refresh Token"`
}
