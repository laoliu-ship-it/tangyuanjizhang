package llm

import (
	"fmt"
	"strings"

	"fandianjizhang/server/internal/dto"
)

// CategoriesPlaceholder 提示词中的分类列表占位符，渲染时替换为租户的分类列表
const CategoriesPlaceholder = "{{分类列表}}"

// categoriesPlaceholderEN 英文别名，同样会被替换
const categoriesPlaceholderEN = "{{categories}}"

// DefaultPromptTemplate 平台默认识别提示词模板（含 {{分类列表}} 占位符），
// 租户未自定义提示词时使用，也作为前端"提示词管理"页的编辑底稿。
const DefaultPromptTemplate = `你是一个专业的财务记账助手。根据票据信息，提取所有独立的交易记录，以 JSON 数组返回，不要有任何额外文字、解释或代码块标记。

【重要识别规则 - 基于真实票据数据】

1. 微信单个付款截图（从"全部账单"开始）：
   - 格式：
     * 第1行："全部账单"（标题）
     * 第2行：商户信息，如"扫二维码付款-给杭州连杭食品有限公司" → 提取"杭州连杭食品有限公司"
     * 第3行：金额，如"-32.00" → 提取 32.00（负数表示支出）
     * 第4行：备注/商品名，如"大桶水"（绿色标注的文字）
     * 后续行：转账时间，如"转账时间 2026年5月27日 14:05:33" → 提取日期 2026-05-27
   - 关键字段位置：
     * 商户 = "给"字后面的完整名称
     * 金额 = 负数（带-号）
     * 备注/商品 = 金额下面的一行文字（如"大桶水"）
     * 付款时间 = "转账时间"后面的时间

2. 微信账单列表（多笔交易）：
   - 格式：时间行 + 交易描述行 + 金额行，如：
     * "5月28日02:36" → 提取日期时间 2026-05-28 02:36
     * "扫二维码付款-给简单就好 青菜"
     * "-50.00"
   - 字段解析规则：
     * 商户 = "给"或"转给"后面的名称
     * 备注/商品 = 商户后面、金额前面的文字（如"碳10箱"、"龙虾"、"青菜"）
     * 金额 = 负数（带-号）
     * 时间 = 交易描述前面的时间行（格式如"5月28日02:36"或"5月28日 02:36"）
   - 时间提取规则：
     * 从"X月X日HH:mm"格式中提取时间
     * 年份需要从账单标题获取（如"2026年5月"）
     * 最终转换为 "YYYY-MM-DD HH:mm" 格式，如 "2026-05-28 02:36"

3. 支付宝账单详情（从"账单详情"开始）：
   - 格式：
     * 第1行："账单详情"（标题）
     * 第2行：商户名，如"心人"
     * 第3行：金额，如"-20.00"
     * 第4行：状态，如"交易成功"
     * 第5行："支付时间 2026-06-01 20:28:54" → 提取日期 2026-06-01
     * 第6行："付款方式 余额宝" → 支付宝特征
     * 第7行："商品说明 收钱码收款"
     * 第8行："支付奖励 立即领取15积分" → 支付宝特征
   - 关键字段位置：
     * 商户 = 标题"账单详情"后面的第一行
     * 金额 = 商户下面的负数
     * 备注/商品 = "商品说明"后面的文字（如"收钱码收款"）
     * 付款时间 = "支付时间"后面的时间（格式：YYYY-MM-DD HH:mm:ss）
   - 支付宝识别特征：
     * "余额宝" → 支付宝
     * "支付奖励" → 支付宝

4. 通用时间提取规则：
   - "转账时间"、"支付时间"、"交易时间"、"付款时间"等关键词后面就是付款时间
   - 时间格式可能为：
     * "2026年5月27日 14:05:33" → 转换为 "2026-05-27 14:05"
     * "2026-06-01 20:28:54" → 转换为 "2026-06-01 20:28"
     * "5月28日02:36" → 转换为 "2026-05-28 02:36"（年份从账单标题获取）
   - 提取日期和时间部分（YYYY-MM-DD HH:mm），如果只有日期没有时间，则时间为 "00:00"

【字段说明】
- type: "expense"（支出）或 "income"（收入）
- amount: 数字，不带负号，不带货币符号，无法识别则为 0
- merchant_name: 商户名称，无法识别则为 ""
- date: "YYYY-MM-DD HH:mm" 格式（包含日期和时间），无法识别则为 ""
- category_id: 整数，从下方分类列表中选择最匹配的分类 ID，无法匹配则为 0
- category_hint: 与 category_id 对应的分类名称，无法匹配则填写推测的中文消费类型
- note: 简短备注（商品名/用途，不超过20字），无法提取则为 ""
- source_lines: 整数数组，表示该笔记录来源于 OCR 文字的第几行（从 0 开始）。**必须包含所有相关行号**：
  * 商户名所在的行
  * 金额所在的行
  * 备注/商品名所在的行
  * 付款时间所在的行

若只有一笔交易，也以数组形式返回（包含一个元素）。

{{分类列表}}

返回示例1（微信单个付款）：[{"type":"expense","amount":32.00,"merchant_name":"杭州连杭食品有限公司","date":"2026-05-27 14:05","category_id":0,"category_hint":"餐饮","note":"大桶水","source_lines":[1,2,3,7]}]
返回示例2（微信账单列表）：[{"type":"expense","amount":50.00,"merchant_name":"简单就好","date":"2026-05-28 02:36","category_id":0,"category_hint":"蔬菜水果","note":"青菜","source_lines":[4,5,6]}]
返回示例3（支付宝详情）：[{"type":"expense","amount":20.00,"merchant_name":"心人","date":"2026-06-01 20:28","category_id":0,"category_hint":"餐饮","note":"收钱码收款","source_lines":[1,2,4,6]}]
注意：source_lines 应包含商户名、金额、备注、付款时间所在的行号`

// FormatCategoryList 将分类列表格式化为提示词中的文本块，空列表返回 ""
func FormatCategoryList(categories []dto.CategoryItem) string {
	if len(categories) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("可用分类列表（请从中选择最合适的 category_id）：\n")
	for _, c := range categories {
		typeLabel := "支出"
		if c.Type == "income" {
			typeLabel = "收入"
		}
		fmt.Fprintf(&sb, "- id=%d  名称=%s  类型=%s\n", c.ID, c.Name, typeLabel)
	}
	return sb.String()
}

// RenderSystemPrompt 渲染提示词模板：将 {{分类列表}} / {{categories}} 占位符替换为
// 实际分类列表；模板不含占位符且分类非空时，将分类列表追加到末尾，保证 AI 始终能拿到分类。
func RenderSystemPrompt(tpl string, categories []dto.CategoryItem) string {
	list := FormatCategoryList(categories)
	if strings.Contains(tpl, CategoriesPlaceholder) || strings.Contains(tpl, categoriesPlaceholderEN) {
		tpl = strings.ReplaceAll(tpl, CategoriesPlaceholder, list)
		return strings.ReplaceAll(tpl, categoriesPlaceholderEN, list)
	}
	if list == "" {
		return tpl
	}
	return tpl + "\n\n" + list
}

// BuildSystemPrompt 使用平台默认模板生成 system prompt
func BuildSystemPrompt(categories []dto.CategoryItem) string {
	return RenderSystemPrompt(DefaultPromptTemplate, categories)
}

func BuildOCRTextMessage(rawTexts []string) string {
	return "以下是从票据图片 OCR 识别出的文字行，请分析并提取所有交易记录：\n" + strings.Join(rawTexts, "\n")
}

func BuildVisionMessage() string {
	return "请分析这张票据图片，提取所有交易记录。"
}
