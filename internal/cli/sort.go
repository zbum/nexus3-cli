package cli

// naturalSort sorts tag strings so that embedded integer runs compare
// numerically (v2 < v10) while the surrounding literals sort lexicographically.
type naturalSort []string

func (s naturalSort) Len() int           { return len(s) }
func (s naturalSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s naturalSort) Less(i, j int) bool { return naturalLess(s[i], s[j]) }

func naturalLess(a, b string) bool {
	for len(a) > 0 && len(b) > 0 {
		ad, an := digitPrefix(a)
		bd, bn := digitPrefix(b)
		if an > 0 && bn > 0 {
			if ad != bd {
				return ad < bd
			}
			a = a[an:]
			b = b[bn:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a = a[1:]
		b = b[1:]
	}
	return len(a) < len(b)
}

func digitPrefix(s string) (value uint64, width int) {
	for width < len(s) && s[width] >= '0' && s[width] <= '9' {
		value = value*10 + uint64(s[width]-'0')
		width++
	}
	return value, width
}
