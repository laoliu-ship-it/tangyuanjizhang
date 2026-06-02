package service

import (
	"context"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

// defaultCategories 餐厅记账场景下的默认分类列表
var defaultCategories = []struct{ Name, Type, Icon string }{
	// 支出
	{"食材采购", "expense", "🥩"},
	{"蔬菜水果", "expense", "🥗"},
	{"海鲜水产", "expense", "🐟"},
	{"酒水饮料", "expense", "🍷"},
	{"燃气水电", "expense", "⚡"},
	{"员工工资", "expense", "👨‍🍳"},
	{"房租", "expense", "🏠"},
	{"清洁耗材", "expense", "🧴"},
	{"餐具包装", "expense", "📦"},
	{"设备维修", "expense", "🔧"},
	{"营销推广", "expense", "📢"},
	{"其他支出", "expense", "💰"},
	// 收入
	{"堂食收入", "income", "🍽️"},
	{"外卖收入", "income", "📱"},
	{"包席预订", "income", "🎉"},
	{"其他收入", "income", "💵"},
}

// createDefaultCategories 为指定租户批量插入默认分类，忽略单条失败
func createDefaultCategories(ctx context.Context, categoryRepo repo.CategoryRepo, tenantID uint64) {
	for _, c := range defaultCategories {
		_ = categoryRepo.Create(ctx, &model.Category{
			TenantID: tenantID,
			Name:     c.Name,
			Type:     c.Type,
			Icon:     c.Icon,
		})
	}
}
