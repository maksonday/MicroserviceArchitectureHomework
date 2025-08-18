package db

import (
	"database/sql"
	"errors"
	"fmt"

	"users/types"

	"github.com/georgysavva/scany/sqlscan"
)

const maxParamLen = 256

var (
	ErrNoUser           = errors.New("user not found")
	ErrUserNameTooLong  = errors.New("username exceeds maximum length")
	ErrFirstNameTooLong = errors.New("first name exceeds maximum length")
	ErrLastNameTooLong  = errors.New("last name exceeds maximum length")
	ErrEmailTooLong     = errors.New("email exceeds maximum length")
	ErrPhoneTooLong     = errors.New("phone exceeds maximum length")
)

func GetUser(id int64) (*types.User, error) {
	rows, err := GetConn().Query(`select username, firstname, lastname, phone, email from users where id = $1 LIMIT 1`, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	defer rows.Close()

	var (
		user  types.User
		count int
	)
	for rows.Next() {
		count++
		err := sqlscan.ScanRow(&user, rows)
		if err != nil {
			return nil, err
		}
	}

	if count == 0 {
		return nil, ErrNoUser
	}

	return &user, rows.Err()
}

func DeleteUser(id int64) error {
	if _, err := GetConn().Exec(`delete from users where id = $1`, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func CreateUser(user *types.User) (int64, error) {
	if err := validateUser(user); err != nil {
		return 0, err
	}

	res := GetConn().QueryRow(`insert into users(username, firstname, lastname, email, phone) values($1, $2, $3, $4, $5) returning id`,
		user.Username, user.FirstName, user.LastName, user.Email, user.Phone)

	var id int64
	if err := res.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("create user: no rows returned")
		}
		return 0, fmt.Errorf("create user: %w", err)
	}

	return id, nil
}

func UpdateUser(user *types.User) error {
	if err := validateUser(user); err != nil {
		return err
	}

	if _, err := GetConn().Exec(`update users set username = $1, firstname = $2, lastname = $3, email = $4, phone = $5 where id = $6`,
		user.Username, user.FirstName, user.LastName, user.Email, user.Phone, user.Id); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func validateUser(user *types.User) error {
	switch {
	case len(user.Username) > maxParamLen:
		return ErrUserNameTooLong
	case len(user.FirstName) > maxParamLen:
		return ErrFirstNameTooLong
	case len(user.LastName) > maxParamLen:
		return ErrLastNameTooLong
	case len(user.Email) > maxParamLen:
		return ErrEmailTooLong
	case len(user.Phone) > maxParamLen:
		return ErrPhoneTooLong
	default:
		return nil
	}
}
