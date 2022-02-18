package pgparty

import (
	"strings"

	"github.com/jmoiron/sqlx"
)

const caps = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func ToSnakeCase2(str string) string {
	out := strings.Builder{}
	out.Grow(len(str) + len(str)/2)

	i := 0
	for i < len(str) {
		b := str[i]
		if strings.IndexByte(caps, b) < 0 {
			out.WriteByte(b)
			i++
			continue
		}
		b += 0x20
		if i > 0 &&
			str[i-1] != '_' &&
			((str[i-1] >= 'a' && str[i-1] <= 'z') ||
				(i+1 < len(str) && str[i+1] >= 'a' && str[i+1] <= 'z')) {
			out.WriteByte('_')
		}
		out.WriteByte(b)
		i++
	}

	return out.String()
}

func init() {
	sqlx.NameMapper = ToSnakeCase2
}
