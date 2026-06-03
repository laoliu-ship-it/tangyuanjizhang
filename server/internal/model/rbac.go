package model

import "time"

// TenantRole 租户角色表
type TenantRole struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64     `gorm:"not null;index" json:"tenant_id"`
	Name      string     `gorm:"size:50;not null" json:"name"`
	IsSystem  bool       `gorm:"not null;default:false" json:"is_system"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`

	Permissions []*RolePermission `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
}

func (TenantRole) TableName() string { return "tenant_roles" }

// RolePermission 角色权限表
type RolePermission struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID    uint64    `gorm:"not null;index" json:"role_id"`
	Resource  string    `gorm:"size:50;not null" json:"resource"`
	Action    string    `gorm:"size:20;not null" json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func (RolePermission) TableName() string { return "role_permissions" }
