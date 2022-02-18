package pgparty

const (
	TagSql       = "sql"
	TagStore     = "store"
	TagKey       = "key"
	TagGinKey    = "ginkey"
	TagLen       = "len"
	TagDBName    = "db"
	TagPrec      = "prec"
	TagDefVal    = "defval"
	TagFullText  = "fulltext"
	TagUniqueKey = "unikey"
	TagPK        = "pk" // `pk:""` - поле входит в первичный ключ, актуально только для новых таблиц

	IDField        = "ID"
	CreatedAtField = "CreatedAt"
	UpdatedAtField = "UpdatedAt"
	DeletedAtField = "DeletedAt"
)
