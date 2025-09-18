package types

type User struct {
	Id        int64  `db:"id" json:"id,omitempty"`
	Username  string `db:"username" json:"username"`
	FirstName string `db:"firstname" json:"firstname"`
	LastName  string `db:"lastname" json:"lastname"`
	Email     string `db:"email" json:"email"`
	Phone     string `db:"phone" json:"phone"`
	Password  string `db:"password" json:"password,omitempty"`
}

type HTTPError struct {
	Error string `json:"error"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}
