package domain

// Norm 把分页参数收束到已发布接口的范围内。
// 省略 page 或 size 时使用 1 和 20。超过 100 的 size 截断为 100，避免已有客户端被直接拒绝。
func Norm(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
