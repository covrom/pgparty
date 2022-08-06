package crud

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	queryObject url.Values
	err         error
}

func NewRequestQueryBuilder() *RequestQueryBuilder {
	rb := &RequestQueryBuilder{
		queryObject: make(url.Values),
	}
	return rb
}

func (rb *RequestQueryBuilder) Query() (string, error) {
	if rb.err != nil {
		return "", rb.err
	}
	if rb.queryObject.Get(DefaultParamNames.Search[0]) != "" {
		rb.queryObject.Del(DefaultParamNames.Filter[0])
		rb.queryObject.Del(DefaultParamNames.Or[0])
	}
	return rb.queryObject.Encode(), nil
}

func (rb *RequestQueryBuilder) UrlValues() (url.Values, error) {
	if rb.err != nil {
		return nil, rb.err
	}
	if rb.queryObject.Get(DefaultParamNames.Search[0]) != "" {
		rb.queryObject.Del(DefaultParamNames.Filter[0])
		rb.queryObject.Del(DefaultParamNames.Or[0])
	}
	return rb.queryObject, nil
}

func (rb *RequestQueryBuilder) Select(fields ...string) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.Fields[0], strings.Join(fields, DelimStr))
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
	rb.queryObject.Set(DefaultParamNames.Search[0], string(b))
	return rb
}

func (rb *RequestQueryBuilder) SetFilter(f ...QueryFilter) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	addQueryObjectParam(rb, DefaultParamNames.Filter[0], f...)
	return rb
}

func addQueryObjectParam[T any](rb *RequestQueryBuilder, name string, f ...T) {
	for _, v := range f {
		rb.queryObject.Add(name, fmt.Sprint(v))
	}
}

func (rb *RequestQueryBuilder) SetOr(f ...QueryFilter) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	addQueryObjectParam(rb, DefaultParamNames.Or[0], f...)
	return rb
}

func (rb *RequestQueryBuilder) SetJoin(j ...QueryJoin) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	addQueryObjectParam(rb, DefaultParamNames.Join[0], j...)
	return rb
}

func (rb *RequestQueryBuilder) SortBy(s ...QuerySort) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	addQueryObjectParam(rb, DefaultParamNames.Sort[0], s...)
	return rb
}

func (rb *RequestQueryBuilder) SetLimit(n int) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.Limit[0], fmt.Sprint(n))
	return rb
}

func (rb *RequestQueryBuilder) SetOffset(n int) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.Offset[0], fmt.Sprint(n))
	return rb
}

func (rb *RequestQueryBuilder) SetPage(n int) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.Page[0], fmt.Sprint(n))
	return rb
}

func (rb *RequestQueryBuilder) ResetCache() *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.Cache[0], "0")
	return rb
}

func (rb *RequestQueryBuilder) SetIncludeDeleted(n int) *RequestQueryBuilder {
	if rb.err != nil {
		return rb
	}
	rb.queryObject.Set(DefaultParamNames.IncludeDeleted[0], fmt.Sprint(n))
	return rb
}

type CreateQueryParams struct {
	Fields         QueryFields
	Search         SCondition
	Filter         []QueryFilter
	Or             []QueryFilter
	Join           []QueryJoin
	Sort           []QuerySort
	Limit          *int
	Offset         *int
	Page           *int
	ResetCache     bool
	IncludeDeleted *int
}

func NewRequestQueryBuilderFromParams(p CreateQueryParams) *RequestQueryBuilder {
	rb := &RequestQueryBuilder{
		queryObject: make(url.Values),
	}
	if p.Fields != nil {
		rb = rb.Select(p.Fields...)
	}
	if p.Search != nil {
		rb = rb.Search(p.Search)
	}
	if p.Filter != nil {
		rb = rb.SetFilter(p.Filter...)
	}
	if p.Or != nil {
		rb = rb.SetOr(p.Or...)
	}
	if p.Join != nil {
		rb = rb.SetJoin(p.Join...)
	}
	if p.Limit != nil {
		rb = rb.SetLimit(*p.Limit)
	}
	if p.Offset != nil {
		rb = rb.SetOffset(*p.Offset)
	}
	if p.Page != nil {
		rb = rb.SetPage(*p.Page)
	}
	if p.Sort != nil {
		rb = rb.SortBy(p.Sort...)
	}
	if p.ResetCache {
		rb = rb.ResetCache()
	}
	if p.IncludeDeleted != nil {
		rb = rb.SetIncludeDeleted(*p.IncludeDeleted)
	}
	return rb
}
