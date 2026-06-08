package llm

import (
	"fmt"
	"strings"

	"fandianjizhang/server/internal/dto"
)

// BuildSystemPrompt 根据分类列表动态生成 system prompt
func BuildSystemPrompt(categories []dto.CategoryItem) string {
	base := `财务记账助手。从票据提取交易，返回纯JSON数组，不含额外文字。

字段：type(expense|income) amount(正数) merchant_name date(YYYY-MM-DD HH:mm,"" 未知) category_id(0未知) category_hint(中文分类) note(≤20字) source_lines(来源行号数组，含商户/金额/备注/时间行)

票据识别规则：
微信单笔："全部账单"→"扫码付款-给{商户}"→负数金额→备注→"转账时间 YYYY年M月D日 HH:mm:ss"
微信列表："{M}月{D}日{HH:mm}"→"扫码付款-给{商户} {备注}"→负数金额（年份从标题取）
支付宝："账单详情"→商户→负数→"交易成功"→"支付时间 YYYY-MM-DD HH:mm:ss"→"商品说明 {备注}"

时间统一输出 YYYY-MM-DD HH:mm，仅有日期时 HH:mm=00:00`

	var sb strings.Builder
	sb.WriteString(base)

	if len(categories) > 0 {
		sb.WriteString("\n分类：")
		for i, c := range categories {
			if i > 0 {
				sb.WriteByte(' ')
			}
			typeLabel := "支"
			if c.Type == "income" {
				typeLabel = "收"
			}
			sb.WriteString(fmt.Sprintf("%d:%s(%s)", c.ID, c.Name, typeLabel))
		}
	}

	sb.WriteString("\n示例：[{\"type\":\"expense\",\"amount\":32.00,\"merchant_name\":\"连杭食品\",\"date\":\"2026-05-27 14:05\",\"category_id\":0,\"category_hint\":\"餐饮\",\"note\":\"大桶水\",\"source_lines\":[1,2,3,7]}]")
	return sb.String()
}

func BuildOCRTextMessage(rawTexts []string) string {
	return "以下是从票据图片 OCR 识别出的文字行，请分析并提取所有交易记录：\n" + strings.Join(rawTexts, "\n")
}

func BuildVisionMessage() string {
	return "请分析这张票据图片，提取所有交易记录。"
}
