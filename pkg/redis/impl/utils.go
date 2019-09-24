package impl

const (
	KeyNotFoundErrMsg = "redis: nil"
)

func isKeyNotFoundError(err error) bool {
	if err.Error() == KeyNotFoundErrMsg {
		return true
	}
	return false
}

func buildHScanFilter(namespace string, filter string) string {
	newFilter := namespace
	if newFilter == "" {
		newFilter = "*"
	}
	newFilter += "/"
	if filter == "" {
		newFilter += "*"
	} else {
		newFilter += filter
	}
	return newFilter
}
