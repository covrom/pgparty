package pgparty

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func (sr *Store) AnalyzeAndReplaceQuery(query, schema string) (string, map[string]ReplaceEntry, error) {
	qps := scanParamsAndQueries(query)
	repl := make([]string, len(qps))

	qrpls := sr.QueryReplacers()

	// выделим только актуально используемые модели
	rpls := make(map[string]ReplaceEntry)
	for _, qp := range qps {
		if qp.param != "" {
			if v, ok := qrpls[qp.param]; ok {
				for kk, vv := range v {
					if _, ok := rpls[kk]; !ok {
						rpls[kk] = vv
					}
				}
			}
		}
	}

	// log.Log.Debugf("qrpls: %v", qrpls)
	// log.Log.Debugf("rpls: %v", rpls)
	// log.Log.Debugf("qps: %v", qps)

	schemapfx := schema + "."
	// сделаем замены
	for i, qp := range qps {
		if qp.param == "" {
			repl[i] = qp.query
			continue
		}
		prm := qp.param
		pfx := ""
		if len(prm) > 13 && prm[:13] == "&CURRSCHEMA.&" {
			pfx = schemapfx
			prm = prm[12:]
		}
		if mdto, ok := rpls[prm]; ok {
			torpl := mdto.To
			if pfx == "" && schema != "" && prm[0] == '&' &&
				qp.query[0] == '&' && !strings.HasPrefix(torpl, schemapfx) && mdto.Schema == "" {
				pfx = schemapfx
			}
			repl[i] = fmt.Sprintf("%s%s", pfx, strings.ReplaceAll(qp.query, prm, torpl))
		} else {
			// оставляем неизменным, если не нашли в заменах
			repl[i] = fmt.Sprintf("%s%s", pfx, qp.query)
		}
	}
	// log.Log.Debugf("repl: %v", repl)
	return strings.Join(repl, ""), rpls, nil
}

// isSpace reports whether the character is a Unicode white space character.
// We avoid dependency on the unicode package, but check validity of the implementation
// in the tests.
func isSpace(r rune) bool {
	if r <= '\u00FF' {
		// Obvious ASCII ones: \t through \r plus space. Plus two Latin-1 oddballs.
		switch r {
		case '&', ':', '.', '_', '*':
			return false
		case '\u0085', '\u00A0':
			return true
		}
		if r < '0' || (r > '9' && r < 'A') || (r > 'Z' && r < 'a') || (r > 'z') {
			return true
		}
		return false
	}
	// High-valued ones.
	if '\u2000' <= r && r <= '\u200a' {
		return true
	}
	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000':
		return true
	}
	return false
}

type scanQP struct {
	query string
	param string
}

func scanParamsAndQueries(query string) []scanQP {
	sb := &strings.Builder{}

	ret := make([]scanQP, 0, 32)
	for len(query) > 0 {
		n, wrd := scanWord(query)
		w := ""
		if wrd != "" && n <= len(query) {
			// в случае алиаса слово может начинаться раньше чем :
			idx := strings.IndexAny(wrd, ":&")
			if idx > 0 {
				wrd = wrd[idx:]
			}
			if (strings.HasPrefix(wrd, "&") || strings.HasPrefix(wrd, ":")) && !strings.HasPrefix(wrd, "::") {
				if idx := strings.Index(wrd, "::"); idx > 0 {
					wrd = wrd[:idx]
				}
				if len(wrd) > 1 {
					w = wrd
				}
			}
		}
		qq := query[:n]
		sb.Reset()
		start := 0
		lastSp := false
		wasQuota1, wasQuota2, wasComment := false, false, false
		for width := 0; start < len(qq); start += width {
			var r rune
			r, width = utf8.DecodeRuneInString(qq[start:])
			if wasComment && r != '\n' {
				continue
			}
			if wasComment && r == '\n' {
				wasComment = false
				continue
			}

			if r == '"' {
				// до следующей кавычки - все считаем пробелом
				if wasQuota2 {
					wasQuota2 = false
					sb.WriteRune(r)
					continue
				}
				if !wasQuota1 {
					wasQuota2 = true
					sb.WriteRune(r)
					continue
				}
			}
			if r == '\'' {
				// до следующей кавычки - все считаем пробелом
				if wasQuota1 {
					wasQuota1 = false
					sb.WriteRune(r)
					continue
				}
				if !wasQuota2 {
					wasQuota1 = true
					sb.WriteRune(r)
					continue
				}
			}
			if wasQuota1 || wasQuota2 {
				sb.WriteRune(r)
				continue
			}
			if (r == '-') && ((start + width) < len(qq)) {
				r2, _ := utf8.DecodeRuneInString(qq[start+width:])
				if r2 == '-' {
					wasComment = true
					continue
				}
			}
			if r < ' ' {
				if !lastSp {
					lastSp = true
					sb.WriteByte(' ')
				}
			} else {
				sb.WriteRune(r)
				lastSp = false
			}
		}
		qq = sb.String()
		ret = append(ret, scanQP{qq, w})
		query = query[n:]
	}
	return ret
}

func scanWord(data string) (advance int, token string) {
	// Skip leading spaces.
	start := 0
	wasQuota1, wasQuota2, wasComment := false, false, false
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRuneInString(data[start:])

		if wasComment && r != '\n' {
			continue
		}
		if wasComment && r == '\n' {
			return start + width, ""
		}

		if r == '"' {
			// до следующей кавычки - все считаем пробелом
			if wasQuota2 {
				wasQuota2 = false
				continue
			}
			if !wasQuota1 {
				wasQuota2 = true
				continue
			}
		}
		if r == '\'' {
			// до следующей кавычки - все считаем пробелом
			if wasQuota1 {
				wasQuota1 = false
				continue
			}
			if !wasQuota2 {
				wasQuota1 = true
				continue
			}
		}
		if wasQuota1 || wasQuota2 {
			continue
		}
		if (r == '-') && ((start + width) < len(data)) {
			r2, _ := utf8.DecodeRuneInString(data[start+width:])
			if r2 == '-' {
				wasComment = true
				continue
			}
		}

		if !isSpace(r) {
			break
		}
	}
	// Scan until space, marking end of word.
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRuneInString(data[i:])
		if isSpace(r) {
			return i + width, data[start:i]
		}
	}
	if len(data) > start {
		return len(data), data[start:]
	}
	// Request more data.
	return start, ""
}
