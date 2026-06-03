package roledefs

// RolePerm 定义单个权限对
type RolePerm struct {
	Resource string
	Action   string
}

// BuiltInRolePerms 返回系统内置角色的权限定义
func BuiltInRolePerms() map[string][]RolePerm {
	return map[string][]RolePerm{
		"admin": {
			{Resource: "*", Action: "*"},
		},
		"finance": {
			{Resource: "transaction", Action: "write"},
			{Resource: "transaction", Action: "read"},
			{Resource: "category", Action: "read"},
			{Resource: "merchant", Action: "read"},
			{Resource: "statistics", Action: "read"},
			{Resource: "export", Action: "read"},
		},
		"partner": {
			{Resource: "transaction", Action: "read"},
			{Resource: "category", Action: "read"},
			{Resource: "merchant", Action: "read"},
			{Resource: "statistics", Action: "read"},
			{Resource: "export", Action: "read"},
		},
	}
}
