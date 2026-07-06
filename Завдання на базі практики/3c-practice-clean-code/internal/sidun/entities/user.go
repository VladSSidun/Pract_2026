package entities

import "errors"

// User — сутність користувача системи
type User struct {
	id       int
	name     string
	password string
	role     string
}

func NewUser(id int, name, password, role string) (*User, error) {
	if name == "" {
		return nil, errors.New("ім'я користувача не може бути порожнім")
	}
	if password == "" {
		return nil, errors.New("пароль не може бути порожнім")
	}
	return &User{id: id, name: name, password: password, role: role}, nil
}

func (u *User) ID() int       { return u.id }
func (u *User) Name() string  { return u.name }
func (u *User) Role() string  { return u.role }

func (u *User) VerifyPassword(candidate string) bool {
	return u.password == candidate
}
