package service

import (
	"errors"
	"log/slog"
	"strings"

	"groundclearance/internal/constants"
	"groundclearance/internal/model"
	"groundclearance/internal/repository"
	"groundclearance/internal/util"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo   *repository.UserRepository
	logger *slog.Logger
}

func NewUserService(repo *repository.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, logger: logger}
}

func (s *UserService) Register(phone, password, name, role string) (*model.User, error) {
	phone = strings.TrimSpace(phone)
	name = strings.TrimSpace(name)
	if phone == "" || name == "" {
		return nil, util.NewAppError(constants.CodeValidationFailed, "phone and name are required")
	}
	if !constants.In(constants.UserRoleValues, role) {
		return nil, util.NewAppError(constants.CodeValidationFailed, "invalid role")
	}
	if _, err := s.repo.FindByPhone(phone); err == nil {
		return nil, util.NewAppError(constants.CodeUserExists, constants.MsgPhoneExists)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Phone: phone, PasswordHash: string(hash), Name: name, Role: role}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogUserRegisterSuccess, "user_id", user.ID)
	return user, nil
}

func (s *UserService) Login(secret string, expireHours int, phone, password string) (string, *model.User, error) {
	user, err := s.repo.FindByPhone(strings.TrimSpace(phone))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.logger.Warn(constants.LogUserLoginFailed, "phone", phone)
		return "", nil, util.NewAppError(constants.CodeInvalidCredentials, constants.MsgInvalidCredentials)
	}
	token, err := util.GenerateToken(secret, util.DurationHours(expireHours), user.ID, user.Phone, user.Role)
	if err != nil {
		return "", nil, err
	}
	s.logger.Info(constants.LogUserLoginSuccess, "user_id", user.ID)
	return token, user, nil
}

func (s *UserService) GetByID(id uint64) (*model.User, error) { return s.repo.FindByID(id) }

func (s *UserService) UpdateProfile(id uint64, name string) (*model.User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	user.Name = name
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogUserProfileUpdate, "user_id", id)
	return user, nil
}

func (s *UserService) List(page, pageSize int) ([]model.User, int64, error) {
	return s.repo.List(page, pageSize)
}
