package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionExpired     = errors.New("session expired or invalid")
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*User, error) {
	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	role := "user"
	count, err := s.repo.UserCount(ctx)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		role = "admin"
	}

	return s.repo.CreateUser(ctx, req.Email, string(hash), req.Name, role)
}

func (s *Service) Login(ctx context.Context, req LoginRequest, userAgent, ip string) (*User, *TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}
	if user.Status != "active" {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.createTokenPair(ctx, user, userAgent, ip)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken, userAgent, ip string) (*TokenPair, error) {
	session, err := s.repo.GetSessionByToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionExpired
	}

	if err := s.repo.DeleteSession(ctx, refreshToken); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return s.createTokenPair(ctx, user, userAgent, ip)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.repo.DeleteSession(ctx, refreshToken)
}

func (s *Service) GetUser(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.ListUsers(ctx, page, size)
}

func (s *Service) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) error {
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 12)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		hashed := string(hash)
		req.Password = &hashed
	}
	return s.repo.UpdateUser(ctx, id, req)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *Service) createTokenPair(ctx context.Context, user *User, userAgent, ip string) (*TokenPair, error) {
	accessToken, err := GenerateAccessToken(s.jwtSecret, user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if _, err := s.repo.CreateSession(ctx, user.ID, refreshToken, userAgent, ip); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
