package user

import (
	"context"
	"net"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	RoleAdmin             = "admin"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Status       Status
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRole struct {
	UserID    string
	RoleCode  string
	GrantedAt time.Time
	GrantedBy *string
}

type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	CreatedAt  time.Time
	LastSeenAt *time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	IP         net.IP
	UserAgent  string
}

type Role struct {
	Code string
	Name string
}

type UserRepository interface {
	Create(context.Context, User) (string, error)
	CreateWithRoles(context.Context, User, []UserRole, bool) (string, error)
	FindByID(context.Context, string) (User, error)
	FindByEmail(context.Context, string) (User, error)
	List(context.Context, string, int, int) ([]User, int64, error)
	Update(context.Context, User) error
	Delete(context.Context, string) error
}

type RoleRepository interface {
	Create(context.Context, Role) (string, error)
	FindByCode(context.Context, string) (Role, error)
	List(context.Context) ([]Role, error)
}

type UserRoleRepository interface {
	Grant(context.Context, UserRole) error
	Revoke(context.Context, string, string) error
	Replace(context.Context, string, []UserRole) error
	ListByUserID(context.Context, string) ([]UserRole, error)
}

type SessionRepository interface {
	Create(context.Context, Session) (string, error)
	FindByTokenHash(context.Context, string) (Session, error)
	Touch(context.Context, string, time.Time, time.Time) error
	Revoke(context.Context, string, time.Time) error
	RevokeAllByUserID(context.Context, string, time.Time) error
}
