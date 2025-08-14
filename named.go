package ygggo_mysql

import (
	"fmt"
	"reflect"
	"strings"
)

// parseNamed converts SQL with :name placeholders to positional ? and returns ordered names.
// Very simple parser: scans runes, recognizes :identifier sequences outside quotes.
func parseNamed(query string) (bound string, names []string) {
	var b strings.Builder
	b.Grow(len(query))
	inSingle, inDouble := false, false
	i := 0
	for i < len(query) {
		ch := query[i]
		switch ch {
		case '\'':
			inSingle = !inSingle && !inDouble
			b.WriteByte(ch)
			i++
			continue
		case '"':
			inDouble = !inDouble && !inSingle
			b.WriteByte(ch)
			i++
			continue
		case ':':
			if inSingle || inDouble {
				b.WriteByte(ch)
				i++
				continue
			}
			// capture identifier
			j := i + 1
			for j < len(query) {
				c := query[j]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
					j++
					continue
				}
				break
			}
			if j > i+1 {
				name := query[i+1 : j]
				names = append(names, name)
				b.WriteByte('?')
				i = j
				continue
			}
			// lone ':'
		}
		b.WriteByte(ch)
		i++
	}
	return b.String(), names
}

// structOrMapToMap flattens a struct (using `db` tags) or passes map[string]any.
func structOrMapToMap(v any) (map[string]any, error) {
	switch m := v.(type) {
	case map[string]any:
		return m, nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer { rv = rv.Elem() }
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or map, got %T", v)
	}
	rt := rv.Type()
	out := make(map[string]any, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		name := f.Tag.Get("db")
		if name == "-" { continue }
		if name == "" { name = strings.ToLower(f.Name) }
		out[name] = rv.Field(i).Interface()
	}
	return out, nil
}

func valuesByNames(m map[string]any, names []string) []any {
	args := make([]any, len(names))
	for i, n := range names {
		args[i] = m[n]
	}
	return args
}

func bindNamed(query string, arg any) (string, []any, error) {
	bound, names := parseNamed(query)
	m, err := structOrMapToMap(arg)
	if err != nil { return "", nil, err }
	args := valuesByNames(m, names)
	return bound, args, nil
}

// dummyResult is a minimal sql.Result implementation used for multi-row NamedExec.
type dummyResult int64

func (d dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (d dummyResult) RowsAffected() (int64, error) { return int64(d), nil }

