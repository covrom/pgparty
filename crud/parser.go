package crud

import (
	"fmt"
	"net/url"

	jsoniter "github.com/json-iterator/go"
)

// JsonQuery syntax: https://github.com/nestjsx/crud/wiki/Requests#query-params
type RequestQuery struct {
	Fields         QueryFields
	ParamsFilter   []QueryFilter
	Search         SCondition
	Filter         []QueryFilter
	Or             []QueryFilter
	Join           []QueryJoin
	Sort           []QuerySort
	Limit          *int
	Offset         *int
	Page           *int
	Cache          *int
	IncludeDeleted *int
}

//	if err := r.ParseForm(); err != nil {
//		return err
//	}
//
// for k, v := range r.Form {
func (p *RequestQuery) ParseQuery(query url.Values) error {
	var err error
	searchData := query.Get(DefaultParamNames.Search[0])
	p.Search, err = p.parseSearchQueryParam(searchData)
	if err != nil {
		return err
	}
	return nil
}

func (p *RequestQuery) parseSearchQueryParam(d string) (SCondition, error) {
	if d == "" {
		return nil, nil
	}
	return p.parseCondition(jsoniter.Get([]byte(d)))
}

func (p *RequestQuery) parseCondition(val jsoniter.Any) (SCondition, error) {
	// search: {$and: [...], ...}
	js := val.Get("$and")
	if js.LastError() == nil {
		if js.ValueType() != jsoniter.ArrayValue {
			return nil, fmt.Errorf("$and: %q is not an array", js.GetInterface())
		}
		sz := js.Size()
		// search: {$and: [{}, {}, ...]}
		andArr := make([]SCondition, 0, sz)

		for i := 0; i < sz; i++ {
			v, err := p.parseCondition(js.Get(i))
			if err != nil {
				return nil, fmt.Errorf("$and[%d] error: %w", i, err)
			}
			andArr = append(andArr, v)
		}
		return CAnd(andArr...), nil
	}
	keys := val.Keys()
	// search: {$or: [...], ...}
	js = val.Get("$or")
	if js.LastError() == nil {
		if js.ValueType() != jsoniter.ArrayValue {
			return nil, fmt.Errorf("$or: %q is not an array", js.GetInterface())
		}
		sz := js.Size()
		orArr := make([]SCondition, 0, sz)
		for i := 0; i < sz; i++ {
			v, err := p.parseCondition(js.Get(i))
			if err != nil {
				return nil, fmt.Errorf("$or[%d] error: %w", i, err)
			}
			orArr = append(orArr, v)
		}
		// search: {$or: [...], foo, ...}
		andArr := make([]SCondition, 0, len(keys))
		for _, field := range keys {
			if field != "$or" {
				jsField := val.Get(field)
				v, err := p.parseCondition(jsField)
				if err != nil {
					return nil, fmt.Errorf("$and[%q] error: %w", field, err)
				}
				andArr = append(andArr, v)
			} else {
				andArr = append(andArr, COr(orArr...))
			}
		}
		return CAnd(andArr...), nil
	}
	// search: {...}
	andArr := make([]SCondition, 0, len(keys))
	for _, field := range keys {
		jsField := val.Get(field)
		v, err := p.parseCondition(jsField)
		if err != nil {
			return nil, fmt.Errorf("$and[%q] error: %w", field, err)
		}
		andArr = append(andArr, v)
	}
	return CAnd(andArr...), nil
}
