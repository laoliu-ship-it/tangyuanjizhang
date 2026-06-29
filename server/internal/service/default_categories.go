package service

import (
	"context"

	"fandianjizhang/server/internal/model"
	"fandianjizhang/server/internal/repo"
)

type categoryDef struct{ Name, Type, Icon string }

var defaultCategoriesMap = map[string][]categoryDef{
	model.BookTypeRestaurant: restaurantCategories,
	model.BookTypeFamily:     familyCategories,
	model.BookTypeStartup:    startupCategories,
}

var restaurantCategories = []categoryDef{
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
	{"堂食收入", "income", "🍽️"},
	{"外卖收入", "income", "📱"},
	{"包席预订", "income", "🎉"},
	{"其他收入", "income", "💵"},
}

var familyCategories = []categoryDef{
	{"餐饮伙食", "expense", "🍱"},
	{"购物消费", "expense", "🛍️"},
	{"交通出行", "expense", "🚗"},
	{"水电煤气", "expense", "💡"},
	{"教育学习", "expense", "📚"},
	{"医疗健康", "expense", "🏥"},
	{"娱乐休闲", "expense", "🎮"},
	{"服装美容", "expense", "👗"},
	{"房租房贷", "expense", "🏠"},
	{"日用品", "expense", "🧹"},
	{"人情往来", "expense", "🎁"},
	{"其他支出", "expense", "💰"},
	{"工资收入", "income", "💼"},
	{"兼职收入", "income", "🤝"},
	{"投资理财", "income", "📈"},
	{"其他收入", "income", "💵"},
}

var startupCategories = []categoryDef{
	{"人力成本", "expense", "👥"},
	{"办公租金", "expense", "🏢"},
	{"设备采购", "expense", "💻"},
	{"营销推广", "expense", "📢"},
	{"差旅费用", "expense", "✈️"},
	{"水电网络", "expense", "🌐"},
	{"软件服务", "expense", "☁️"},
	{"税费", "expense", "🧾"},
	{"外包服务", "expense", "🔗"},
	{"其他支出", "expense", "💰"},
	{"产品销售", "income", "📦"},
	{"服务收入", "income", "🤝"},
	{"融资收入", "income", "💰"},
	{"投资收益", "income", "📈"},
	{"其他收入", "income", "💵"},
}

// createDefaultCategories 为指定租户按账本类型批量插入默认分类，忽略单条失败
func createDefaultCategories(ctx context.Context, categoryRepo repo.CategoryRepo, tenantID uint64, bookType string) {
	cats, ok := defaultCategoriesMap[bookType]
	if !ok {
		cats = restaurantCategories
	}
	for _, c := range cats {
		_ = categoryRepo.Create(ctx, &model.Category{
			TenantID: tenantID,
			Name:     c.Name,
			Type:     c.Type,
			Icon:     c.Icon,
		})
	}
}
