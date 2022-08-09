package crud

import (
	"fmt"
	"reflect"
	"strings"
)

type QueryFields = []string

type QueryFilter struct {
	Field    string
	Operator ComparisonOperator
	Value    any
}

func (f QueryFilter) String() string {
	if f.Value == nil {
		return fmt.Sprintf("%s%s%s", f.Field, Delim, f.Operator)
	}
	return fmt.Sprintf("%s%s%s%s%s", f.Field, Delim, f.Operator, Delim, f.Value)
}

type QueryJoin struct {
	Field  string
	Select QueryFields
}

func (j QueryJoin) String() string {
	if len(j.Select) == 0 {
		return j.Field
	}
	return fmt.Sprintf("%s%s%s", j.Field, Delim, strings.Join(j.Select, DelimStr))
}

type QuerySort struct {
	Field string
	Order QuerySortOperator
}

func (s QuerySort) String() string {
	return fmt.Sprintf("%s%s%s", s.Field, DelimStr, s.Order)
}

type QuerySortOperator string

var (
	SORT_ASC  QuerySortOperator = "ASC"
	SORT_DESC QuerySortOperator = "DESC"
)

type ComparisonOperator string

const (
	EQUALS              ComparisonOperator = "$eq"
	NOT_EQUALS          ComparisonOperator = "$ne"
	GREATER_THAN        ComparisonOperator = "$gt"
	LOWER_THAN          ComparisonOperator = "$lt"
	GREATER_THAN_EQUALS ComparisonOperator = "$gte"
	LOWER_THAN_EQUALS   ComparisonOperator = "$lte"
	STARTS              ComparisonOperator = "$starts"
	ENDS                ComparisonOperator = "$ends"
	CONTAINS            ComparisonOperator = "$cont"
	EXCLUDES            ComparisonOperator = "$excl"
	IN                  ComparisonOperator = "$in"
	NOT_IN              ComparisonOperator = "$notin"
	IS_NULL             ComparisonOperator = "$isnull"
	NOT_NULL            ComparisonOperator = "$notnull"
	BETWEEN             ComparisonOperator = "$between"
	EQUALS_LOW          ComparisonOperator = "$eqL"
	NOT_EQUALS_LOW      ComparisonOperator = "$neL"
	STARTS_LOW          ComparisonOperator = "$startsL"
	ENDS_LOW            ComparisonOperator = "$endsL"
	CONTAINS_LOW        ComparisonOperator = "$contL"
	EXCLUDES_LOW        ComparisonOperator = "$exclL"
	IN_LOW              ComparisonOperator = "$inL"
	NOT_IN_LOW          ComparisonOperator = "$notinL"
)

// type SCondition struct {
// 	SFields
// 	SConditionAND
// }

type queryNode interface {
	queryNode()
}

type qNode struct{}

func (qNode) queryNode() {}

type SPrimitivesVal interface {
	queryNode
	primitive()
}

type String string

func (String) queryNode()  {}
func (String) primitive()  {}
func (String) fieldValue() {}
func (String) field()      {}

type Int int64

func (Int) queryNode()  {}
func (Int) primitive()  {}
func (Int) fieldValue() {}
func (Int) field()      {}

type Float float64

func (Float) queryNode()  {}
func (Float) primitive()  {}
func (Float) fieldValue() {}
func (Float) field()      {}

type Bool bool

func (Bool) queryNode()  {}
func (Bool) primitive()  {}
func (Bool) fieldValue() {}
func (Bool) field()      {}

type StringArray []string

func (StringArray) queryNode()  {}
func (StringArray) fieldValue() {}
func (StringArray) field()      {}

type IntArray []int64

func (IntArray) queryNode()  {}
func (IntArray) fieldValue() {}
func (IntArray) field()      {}

type FloatArray []float64

func (FloatArray) queryNode()  {}
func (FloatArray) fieldValue() {}
func (FloatArray) field()      {}

type BoolArray []bool

func (BoolArray) queryNode()  {}
func (BoolArray) fieldValue() {}
func (BoolArray) field()      {}

type SFiledValues interface {
	queryNode
	fieldValue()
}

type SFieldOperator struct {
	Eq      SFiledValues    `json:"$eq,omitempty"`
	Ne      SFiledValues    `json:"$ne,omitempty"`
	Gt      SFiledValues    `json:"$gt,omitempty"`
	Lt      SFiledValues    `json:"$lt,omitempty"`
	Gte     SFiledValues    `json:"$gte,omitempty"`
	Lte     SFiledValues    `json:"$lte,omitempty"`
	Starts  SFiledValues    `json:"$starts,omitempty"`
	Ends    SFiledValues    `json:"$ends,omitempty"`
	Cont    SFiledValues    `json:"$cont,omitempty"`
	Excl    SFiledValues    `json:"$excl,omitempty"`
	In      SFiledValues    `json:"$in,omitempty"`
	Notin   SFiledValues    `json:"$notin,omitempty"`
	Between SFiledValues    `json:"$between,omitempty"`
	Isnull  SFiledValues    `json:"$isnull,omitempty"`
	Notnull SFiledValues    `json:"$notnull,omitempty"`
	EqL     SFiledValues    `json:"$eqL,omitempty"`
	NeL     SFiledValues    `json:"$neL,omitempty"`
	StartsL SFiledValues    `json:"$startsL,omitempty"`
	EndsL   SFiledValues    `json:"$endsL,omitempty"`
	ContL   SFiledValues    `json:"$contL,omitempty"`
	ExclL   SFiledValues    `json:"$exclL,omitempty"`
	InL     SFiledValues    `json:"$inL,omitempty"`
	NotinL  SFiledValues    `json:"$notinL,omitempty"`
	Or      *SFieldOperator `json:"$or,omitempty"`
	// $and?: never;
}

func (SFieldOperator) queryNode() {}
func (SFieldOperator) field()     {}

type SField interface {
	queryNode
	field()
}

// type SFields = {
// 	[key: string]: SField | Array<SFields | SConditionAND> | undefined;
// 	$or?: Array<SFields | SConditionAND>;
// 	$and?: never;
//   }

type SFields map[string]interface{}

func (SFields) queryNode() {}
func (SFields) cond()      {}
func (fs SFields) validate() error {
	for k, v := range fs {
		switch k {
		case "$and":
			return fmt.Errorf("$and forbiden in fields")
		case "$or":
			if err := isArraySFieldsOrSConditionAND(v); err != nil {
				return err
			}
		default:
			if v != nil {
				if err := isArraySFieldsOrSConditionAND(v); err != nil {
					_, ok := v.(SField)
					if !ok {
						return fmt.Errorf("%T is not a field, array or slice", v)
					}
				}
			}
		}
	}
	return nil
}

func isArraySFieldsOrSConditionAND(v interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			elt := rv.Index(i).Type()
			if elt == reflect.TypeOf(SConditionAND{}) || elt == reflect.TypeOf(SFields{}) {
				continue
			}
			return fmt.Errorf("elem at index %d: type %s is not SConditionAND or SFields", i, elt)
		}
	}
	return fmt.Errorf("%T is not an array or slice", v)
}

//	type SConditionAND = {
//		$and?: Array<SFields | SConditionAND>;
//		$or?: never;
//	  }
type SConditionAND struct {
	And []SCondition `json:"$and,omitempty"`
}

func (SConditionAND) queryNode() {}
func (SConditionAND) cond()      {}

// type SCondition = SFields | SConditionAND;
type SCondition interface {
	queryNode
	cond()
}

func CAnd(ands ...SCondition) SCondition {
	if len(ands) == 1 {
		return ands[0]
	}
	return SConditionAND{
		And: ands,
	}
}

func COr(ors ...SCondition) SCondition {
	if len(ors) == 1 {
		return ors[0]
	}
	return SFields{
		"$or": ors,
	}
}
