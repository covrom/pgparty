package crud

type QueryFields = []string

type QueryFilter struct {
	Field    string
	Operator ComparisonOperator
	Value    any
}

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

type SFields interface {
	queryNode
	fields()
}

type SFieldsMapField map[string]SField

func (SFieldsMapField) queryNode() {}
func (SFieldsMapField) fields()    {}
func (SFieldsMapField) cond()      {}

type SFieldsMapFields map[string][]SFields

func (SFieldsMapFields) queryNode() {}
func (SFieldsMapFields) fields()    {}
func (SFieldsMapFields) cond()      {}

type SFieldsMapConditionAnd map[string][]SConditionAND

func (SFieldsMapConditionAnd) queryNode() {}
func (SFieldsMapConditionAnd) fields()    {}
func (SFieldsMapConditionAnd) cond()      {}

type SFieldsOrFields struct {
	Or []SFields `json:"$or,omitempty"`
}

func (SFieldsOrFields) queryNode() {}
func (SFieldsOrFields) fields()    {}
func (SFieldsOrFields) cond()      {}

type SFieldsOrConditionAnd struct {
	Or []SConditionAND `json:"$or,omitempty"`
}

func (SFieldsOrConditionAnd) queryNode() {}
func (SFieldsOrConditionAnd) fields()    {}
func (SFieldsOrConditionAnd) cond()      {}

// type SConditionAND = {
// 	$and?: Array<SFields | SConditionAND>;
// 	$or?: never;
//   }

type SConditionAND interface {
	queryNode
	and()
}

type SConditionANDFields struct {
	And []SFields `json:"$and,omitempty"`
}

func (SConditionANDFields) queryNode() {}
func (SConditionANDFields) and()       {}
func (SConditionANDFields) cond()      {}

type SConditionANDConditionAnd struct {
	And []SConditionAND `json:"$and,omitempty"`
}

func (SConditionANDConditionAnd) queryNode() {}
func (SConditionANDConditionAnd) and()       {}
func (SConditionANDConditionAnd) cond()      {}

type SCondition interface {
	queryNode
	cond()
}
