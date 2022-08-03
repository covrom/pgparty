package pgparty

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// JsonQuery syntax: https://github.com/nestjsx/crud/wiki/Requests#query-params
type JsonQuery struct {
	Fields []string `json:"fields,omitempty"`

	S string `json:"s,omitempty"`

	Filter []string `json:"filter,omitempty"`
	Or     []string `json:"or,omitempty"`

	Join string `json:"join,omitempty"`

	Sort []string `json:"sort,omitempty"`

	Limit  *int `json:"limit,omitempty"`
	Offset *int `json:"offset,omitempty"`

	Page *int `json:"page,omitempty"`

	SkipCache bool `json:"skipCache,omitempty"`
}

func (q *JsonQuery) ParseRequest(r *http.Request) error {
	if q == nil {
		panic("JsonQuery pointer is nil")
	}
	*q = JsonQuery{}
	if err := r.ParseForm(); err != nil {
		return err
	}
	for k, v := range r.Form {
		switch strings.ToLower(k) {
		case "fields", "select":
			for _, vv := range v {
				flds := strings.Split(vv, ",")
				for _, fld := range flds {
					q.Fields = append(q.Fields, fld)
				}
			}
		case "s":
			for _, vv := range v {
				q.S = vv
			}
		case "filter":
			for _, vv := range v {
				q.Filter = append(q.Filter, vv)
			}
		case "or":
			for _, vv := range v {
				q.Filter = append(q.Filter, vv)
			}
		case "join":
			for _, vv := range v {
				q.Join = vv
			}
		case "sort":
			for _, vv := range v {
				q.Sort = append(q.Sort, vv)
			}
		case "per_page", "limit":
			for _, vv := range v {
				lim, err := strconv.Atoi(vv)
				if err != nil {
					return fmt.Errorf("limit is not a number")
				}
				q.Limit = &lim
			}
		case "offset":
			for _, vv := range v {
				off, err := strconv.Atoi(vv)
				if err != nil {
					return fmt.Errorf("offset is not a number")
				}
				q.Offset = &off
			}
		case "page":
			for _, vv := range v {
				p, err := strconv.Atoi(vv)
				if err != nil {
					return fmt.Errorf("offset is not a number")
				}
				q.Page = &p
			}
		case "cache":
			q.SkipCache = true
		}
	}
	return nil
}
