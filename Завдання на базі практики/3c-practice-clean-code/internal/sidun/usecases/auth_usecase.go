package usecases

import (
	"errors"
	"fmt"
	"time"
)

// AuthUseCase — логіка автентифікації
type AuthUseCase struct {
	userRepo UserRepository
}

// AuthResult — результат автентифікації (проміжна структура)
type AuthResult struct {
	UserID   int
	UserName string
	Token    string
}

func NewAuthUseCase(repo UserRepository) *AuthUseCase {
	return &AuthUseCase{userRepo: repo}
}

// Login виконує автентифікацію за логіном і паролем
func (uc *AuthUseCase) Login(username, password string) (AuthResult, error) {
	if username == "" || password == "" {
		return AuthResult{}, errors.New("логін і пароль обов'язкові")
	}

	user, err := uc.userRepo.FindByCredentials(username, password)
	if err != nil {
		return AuthResult{}, errors.New("невірні облікові дані")
	}

	token := generateToken(user.ID())

	return AuthResult{
		UserID:   user.ID(),
		UserName: user.Name(),
		Token:    token,
	}, nil
}

func generateToken(userID int) string {
	return fmt.Sprintf("token-%d-%d", userID, time.Now().Unix())
}
