package events

const (
	defaultUpcomingLimit = 4
	maxUpcomingLimit     = 20
	defaultListLimit     = 50
	maxListLimit         = 100
)

func normalizeUpcomingLimit(limit int) int {
	if limit <= 0 {
		return defaultUpcomingLimit
	}
	if limit > maxUpcomingLimit {
		return maxUpcomingLimit
	}
	return limit
}

func normalizeListLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
