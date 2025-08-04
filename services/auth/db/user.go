package db

import (
	"database/sql"
	"errors"
	"fmt"
	"unicode"

	"miniapp/types"

	"github.com/georgysavva/scany/sqlscan"
	"golang.org/x/crypto/bcrypt"
)

const maxParamLen = 256

var (
	ErrNoUser           = errors.New("user not found")
	ErrUserNameTooLong  = errors.New("username exceeds maximum length")
	ErrFirstNameTooLong = errors.New("first name exceeds maximum length")
	ErrLastNameTooLong  = errors.New("last name exceeds maximum length")
	ErrEmailTooLong     = errors.New("email exceeds maximum length")
	ErrPhoneTooLong     = errors.New("phone exceeds maximum length")
	ErrUsernameIsTaken  = errors.New("username is already taken")
	ErrEmailIsTaken     = errors.New("email is already taken")
)

func GetUserCredentials(username string) (*types.User, error) {
	rows, err := GetConn().Query(`select id, email, password from users where username = $1 LIMIT 1`, username)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	defer rows.Close()

	var user types.User
	if rows.Next() {
		if err := sqlscan.ScanRow(&user, rows); err != nil {
			return nil, err
		}
	} else {
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

func hashPassword(password string) (string, error) {
	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	if len(password) >= 7 {
		hasMinLen = true
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	switch {
	case !hasMinLen:
		return "", errors.New("password must be at least 7 characters long")
	case !hasUpper:
		return "", errors.New("password must contain at least one uppercase letter")
	case !hasLower:
		return "", errors.New("password must contain at least one lowercase letter")
	case !hasNumber:
		return "", errors.New("password must contain at least one number")
	case !hasSpecial:
		return "", errors.New("password must contain at least one special character")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func checkUserExists(username, email string) error {
	var user types.User
	err := GetConn().QueryRow(`select username, email from users where username = $1 or email = $2`, username, email).Scan(&user.Username, &user.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("check user existence: %w", err)
	}

	switch {
	case user.Username == username:
		return ErrUsernameIsTaken
	case user.Email == email:
		return ErrEmailIsTaken
	default:
		return nil
	}
}

func CreateUser(user *types.User) (int64, error) {
	if err := validateUser(user); err != nil {
		return 0, err
	}

	if err := checkUserExists(user.Username, user.Email); err != nil {
		return 0, fmt.Errorf("create user: %w", err)
	}

	encPassword, err := hashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	res := GetConn().QueryRow(`insert into users(username, firstname, lastname, email, phone, password) values($1, $2, $3, $4, $5, $6) returning id`,
		user.Username, user.FirstName, user.LastName, user.Email, user.Phone, encPassword)

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
