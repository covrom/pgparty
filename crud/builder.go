package crud

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ParamNames struct {
	Fields         []string
	Search         []string
	Filter         []string
	Or             []string
	Join           []string
	Sort           []string
	Limit          []string
	Offset         []string
	Page           []string
	Cache          []string
	IncludeDeleted []string
}

var (
	Delim             = "||"
	DelimStr          = ","
	DefaultParamNames = &ParamNames{
		Fields:         []string{"fields", "select"},
		Search:         []string{"s"},
		Filter:         []string{"filter"},
		Or:             []string{"or"},
		Join:           []string{"join"},
		Sort:           []string{"sort"},
		Limit:          []string{"limit", "per_page"},
		Offset:         []string{"offset"},
		Page:           []string{"page"},
		Cache:          []string{"cache"},
		IncludeDeleted: []string{"include_deleted"},
	}
)

type RequestQueryBuilder struct {
	queryObject map[string]any
	err         error
}

func NewRequestQueryBuilder() *RequestQueryBuilder {
	rb := &RequestQueryBuilder{
		queryObject: make(map[string]any),
	}
	return rb
}

func (rb *RequestQueryBuilder) Query() (string, error) {
	if rb.err != nil {
		return "", rb.err
	}
	// TODO:
	return "", fmt.Errorf("not implemented")
}

func (rb *RequestQueryBuilder) Select(fields ...string) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject[DefaultParamNames.Fields[0]] = strings.Join(fields, DelimStr)
	return rb
}

func (rb *RequestQueryBuilder) Search(s SCondition) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	b, err := json.Marshal(s)
	if err != nil {
		rb.err = err
		return rb
	}
	rb.queryObject[DefaultParamNames.Search[0]] = string(b)
	return rb
}
