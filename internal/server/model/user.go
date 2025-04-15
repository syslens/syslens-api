package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserRole 定义用户角色类型
type UserRole string

const (
	// RoleAdmin 管理员角色
	RoleAdmin UserRole = "admin"
	// RoleUser 普通用户角色
	RoleUser UserRole = "user"
	// RoleGuest 访客角色
	RoleGuest UserRole = "guest"
)

// User 用户模型
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // 密码散列不应包含在JSON响应中
	Roles        []UserRole `json:"roles"`
	FirstName    string     `json:"first_name,omitempty"`
	LastName     string     `json:"last_name,omitempty"`
	Avatar       string     `json:"avatar,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	IsActive     bool       `json:"is_active"`
}

// NewUser 创建一个新用户
func NewUser(username, email string, roles []UserRole) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New().String(),
		Username:  username,
		Email:     email,
		Roles:     roles,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}
}

// HasRole 检查用户是否具有特定角色
func (u *User) HasRole(role UserRole) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin 检查用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.HasRole(RoleAdmin)
}

// AddRole 为用户添加角色
func (u *User) AddRole(role UserRole) {
	// 检查角色是否已存在
	if u.HasRole(role) {
		return
	}
	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now()
}

// RemoveRole 从用户中删除角色
func (u *User) RemoveRole(role UserRole) {
	var newRoles []UserRole
	for _, r := range u.Roles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}
	u.Roles = newRoles
	u.UpdatedAt = time.Now()
}

// RolesToStrings 将用户角色转换为字符串切片
func (u *User) RolesToStrings() []string {
	roles := make([]string, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = string(role)
	}
	return roles
}

// MarshalJSON 自定义JSON序列化
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		PasswordHash string `json:"-"`
	}{
		Alias:        (*Alias)(u),
		PasswordHash: "",
	})
}

// UpdateActivity 更新用户活动状态
func (u *User) UpdateActivity() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 用户登录响应
type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
	Expiry int64  `json:"expiry"` // 以秒为单位的过期时间戳
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	Password  *string `json:"password,omitempty" binding:"omitempty,min=8"`
}
