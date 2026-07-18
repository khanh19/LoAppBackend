package lists

const (
	defaultPlansLimit = 10
	maxPlansLimit     = 50
)

func normalizePlansLimit(limit int) int {
	if limit <= 0 {
		return defaultPlansLimit
	}
	if limit > maxPlansLimit {
		return maxPlansLimit
	}
	return limit
}
