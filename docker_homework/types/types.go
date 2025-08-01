package types

type User struct {
	Id        int64  `db:"id" json:"id,omitempty"`
	Username  string `db:"username" json:"username"`
	FirstName string `db:"firstname" json:"firstname"`
	LastName  string `db:"lastname" json:"lastname"`
	Email     string `db:"email" json:"email"`
	Phone     string `db:"phone" json:"phone"`
}
