package util

func Cycle(curr string, values []string) string {
	for i, v := range values {
		if v == curr {
			iNext := (i + 1) % len(values)
			return values[iNext]
		}
	}
	if len(values) == 0 {
		return curr
	} else {
		return values[0]
	}
}
