package gqlorm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib" // Postgresql driver
)

// Framework - основной класс фреймворка
type Framework struct {
	db         *sqlx.DB
	schema     *graphql.Schema
	models     map[string]Model
	rbac       *RBAC
	middleware *Middleware
}

// NewFramework создает новый экземпляр фреймворка
func NewFramework(db *sqlx.DB) *Framework {
	return &Framework{
		db:     db,
		models: make(map[string]Model),
		rbac:   NewRBAC(),
	}
}

// Model определяет структуру модели данных
type Model struct {
	Name        string
	Table       string
	Fields      []Field
	Relations   []Relation
	Permissions ModelPermissions
}

// Field описывает поле модели
type Field struct {
	Name       string
	Type       string
	PrimaryKey bool
	Required   bool
}

// Relation описывает связь между моделями
type Relation struct {
	Name       string
	Target     string
	Type       string // "one_to_one", "one_to_many", "many_to_many"
	ForeignKey string
	JoinTable  string // только для many_to_many
}

// ModelPermissions определяет права доступа к модели
type ModelPermissions struct {
	Create []string
	Read   []string
	Update []string
	Delete []string
}

// RBAC управляет правами доступа
type RBAC struct {
	roles map[string]map[string]bool
}

func NewRBAC() *RBAC {
	return &RBAC{
		roles: make(map[string]map[string]bool),
	}
}

// Middleware обрабатывает аутентификацию и авторизацию
type Middleware struct {
	authFunc func(ctx context.Context) (string, []string, error)
}

// RegisterModel регистрирует модель в фреймворке
func (f *Framework) RegisterModel(model Model) error {
	if _, exists := f.models[model.Name]; exists {
		return fmt.Errorf("model %s already registered", model.Name)
	}

	f.models[model.Name] = model
	return nil
}

// Initialize создает GraphQL схему и применяет миграции
func (f *Framework) Initialize() error {
	if err := f.applyMigrations(); err != nil {
		return fmt.Errorf("migrations failed: %v", err)
	}

	schema, err := f.buildGraphQLSchema()
	if err != nil {
		return fmt.Errorf("failed to build GraphQL schema: %v", err)
	}

	f.schema = schema
	return nil
}

// applyMigrations автоматически изменяет схему БД
func (f *Framework) applyMigrations() error {
	for _, model := range f.models {
		if err := f.migrateModel(model); err != nil {
			return err
		}
	}
	return nil
}

// migrateModel создает или изменяет таблицу для модели
func (f *Framework) migrateModel(model Model) error {
	// Проверяем существование таблицы
	var exists bool
	err := f.db.QueryRow(
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
		model.Table,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Создаем новую таблицу
		if err := f.createTable(model); err != nil {
			return err
		}
	} else {
		// Обновляем существующую таблицу
		if err := f.alterTable(model); err != nil {
			return err
		}
	}

	// Создаем таблицы для связей many-to-many
	for _, rel := range model.Relations {
		if rel.Type == "many_to_many" {
			if err := f.createJoinTable(rel); err != nil {
				return err
			}
		}
	}

	return nil
}

// createTable создает новую таблицу
func (f *Framework) createTable(model Model) error {
	columns := []string{}
	for _, field := range model.Fields {
		col := fmt.Sprintf("%s %s", field.Name, f.mapTypeToSQL(field.Type))
		if field.PrimaryKey {
			col += " PRIMARY KEY"
		}
		if field.Required && !field.PrimaryKey {
			col += " NOT NULL"
		}
		columns = append(columns, col)
	}

	query := fmt.Sprintf("CREATE TABLE %s (%s)", model.Table, strings.Join(columns, ", "))
	_, err := f.db.Exec(query)
	return err
}

// alterTable изменяет существующую таблицу
func (f *Framework) alterTable(model Model) error {
	existingColumns := make(map[string]bool)
	rows, err := f.db.Query(
		"SELECT column_name FROM information_schema.columns WHERE table_name = $1",
		model.Table,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return err
		}
		existingColumns[col] = true
	}

	for _, field := range model.Fields {
		if !existingColumns[field.Name] {
			query := fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s %s",
				model.Table,
				field.Name,
				f.mapTypeToSQL(field.Type),
			)
			if _, err := f.db.Exec(query); err != nil {
				return err
			}
		}
	}

	return nil
}

// createJoinTable создает таблицу для связи many-to-many
func (f *Framework) createJoinTable(rel Relation) error {
	// Проверяем существование таблицы
	var exists bool
	err := f.db.QueryRow(
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
		rel.JoinTable,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		query := fmt.Sprintf(
			"CREATE TABLE %s (%s_id UUID, %s_id UUID, PRIMARY KEY (%s_id, %s_id))",
			rel.JoinTable,
			strings.ToLower(rel.Name),
			strings.ToLower(rel.Target),
			strings.ToLower(rel.Name),
			strings.ToLower(rel.Target),
		)
		_, err = f.db.Exec(query)
		return err
	}

	return nil
}

// mapTypeToSQL преобразует тип модели в SQL тип
func (f *Framework) mapTypeToSQL(typ string) string {
	switch typ {
	case "string":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "float":
		return "FLOAT"
	case "bool":
		return "BOOLEAN"
	case "uuid":
		return "UUID"
	case "timestamp":
		return "TIMESTAMP"
	default:
		return "TEXT"
	}
}

// buildGraphQLSchema создает GraphQL схему на основе зарегистрированных моделей
func (f *Framework) buildGraphQLSchema() (*graphql.Schema, error) {
	// Создаем GraphQL типы для всех моделей
	types := make(map[string]*graphql.Object)
	for name, model := range f.models {
		fields := graphql.Fields{}
		for _, field := range model.Fields {
			fields[field.Name] = &graphql.Field{
				Type: f.mapTypeToGraphQL(field.Type),
			}
		}
		types[name] = graphql.NewObject(graphql.ObjectConfig{
			Name:   name,
			Fields: fields,
		})
	}

	// Добавляем связи между типами
	for name, model := range f.models {
		for _, rel := range model.Relations {
			if targetType, exists := types[rel.Target]; exists {
				switch rel.Type {
				case "one_to_one":
					types[name].AddFieldConfig(rel.Name, &graphql.Field{
						Type: targetType,
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return f.resolveRelation(p, rel)
						},
					})
				case "one_to_many":
					types[name].AddFieldConfig(rel.Name, &graphql.Field{
						Type: graphql.NewList(targetType),
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return f.resolveRelation(p, rel)
						},
					})
				case "many_to_many":
					types[name].AddFieldConfig(rel.Name, &graphql.Field{
						Type: graphql.NewList(targetType),
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return f.resolveManyToMany(p, rel)
						},
					})
				}
			}
		}
	}

	// Создаем корневой Query тип
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"get": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(types["User"]))),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: f.resolveGet,
			},
		},
	})

	// Создаем корневой Mutation тип
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"create": &graphql.Field{
				Type: types["User"],
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.NewInputObject(
							graphql.InputObjectConfig{
								Name: "CreateUserInput",
								Fields: graphql.InputObjectConfigFieldMap{
									"name": &graphql.InputObjectFieldConfig{
										Type: graphql.NewNonNull(graphql.String),
									},
								},
							},
						)),
					},
				},
				Resolve: f.resolveCreate,
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// mapTypeToGraphQL преобразует тип модели в GraphQL тип
func (f *Framework) mapTypeToGraphQL(typ string) graphql.Output {
	switch typ {
	case "string":
		return graphql.String
	case "int":
		return graphql.Int
	case "float":
		return graphql.Float
	case "bool":
		return graphql.Boolean
	case "uuid":
		return graphql.String
	case "timestamp":
		return graphql.String
	default:
		return graphql.String
	}
}

// resolveGet обрабатывает GraphQL запрос на получение данных
func (f *Framework) resolveGet(p graphql.ResolveParams) (interface{}, error) {
	// Проверяем права доступа
	if err := f.checkPermissions(p.Context, "read", p.Info.ParentType.Name()); err != nil {
		return nil, err
	}

	model, exists := f.models[p.Info.ParentType.Name()]
	if !exists {
		return nil, fmt.Errorf("model %s not found", p.Info.ParentType.Name())
	}

	// Строим SQL запрос с учетом вложенных полей
	query, args, err := f.buildSelectQuery(model, p.Args, p.Info.FieldASTs)
	if err != nil {
		return nil, err
	}

	// Выполняем запрос
	var result []map[string]interface{}
	if err := f.db.Select(&result, query, args...); err != nil {
		return nil, err
	}

	return result, nil
}

// buildSelectQuery строит SQL запрос с учетом вложенных полей
func (f *Framework) buildSelectQuery(model Model, args map[string]interface{}, fields []*ast.Field) (string, []interface{}, error) {
	var selectFields []string
	var joins []string
	var where []string
	var params []interface{}
	paramCounter := 1

	// Обрабатываем основные поля
	for _, field := range fields {
		if !strings.Contains(field.Name, "__") {
			selectFields = append(selectFields, fmt.Sprintf("%s.%s", model.Table, field.Name))
		}
	}

	// Обрабатываем аргументы фильтрации
	for arg, value := range args {
		if arg == "id" {
			where = append(where, fmt.Sprintf("%s.id = $%d", model.Table, paramCounter))
			params = append(params, value)
			paramCounter++
		}
	}

	// Обрабатываем вложенные поля (связи)
	for _, field := range fields {
		if strings.Contains(field.Name, "__") {
			parts := strings.Split(field.Name, "__")
			if len(parts) != 2 {
				continue
			}

			relationName := parts[0]
			relationField := parts[1]

			// Находим связь
			var relation Relation
			for _, rel := range model.Relations {
				if rel.Name == relationName {
					relation = rel
					break
				}
			}

			if relation.Name == "" {
				continue
			}

			// Добавляем JOIN
			switch relation.Type {
			case "one_to_one", "one_to_many":
				joins = append(joins, fmt.Sprintf(
					"LEFT JOIN %s ON %s.%s = %s.id",
					relation.Target,
					relation.Target,
					relation.ForeignKey,
					model.Table,
				))
				selectFields = append(selectFields, fmt.Sprintf("%s.%s as %s", relation.Target, relationField, field.Name))
			case "many_to_many":
				joins = append(joins, fmt.Sprintf(
					"LEFT JOIN %s ON %s.%s_id = %s.id",
					relation.JoinTable,
					relation.JoinTable,
					strings.ToLower(model.Name),
					model.Table,
				))
				joins = append(joins, fmt.Sprintf(
					"LEFT JOIN %s ON %s.%s_id = %s.id",
					relation.Target,
					relation.JoinTable,
					strings.ToLower(relation.Target),
					relation.Target,
				))
				selectFields = append(selectFields, fmt.Sprintf("%s.%s as %s", relation.Target, relationField, field.Name))
			}
		}
	}

	// Собираем итоговый запрос
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectFields, ", "), model.Table)
	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	return query, params, nil
}

// resolveCreate обрабатывает GraphQL запрос на создание данных
func (f *Framework) resolveCreate(p graphql.ResolveParams) (interface{}, error) {
	// Проверяем права доступа
	if err := f.checkPermissions(p.Context, "create", p.Info.ParentType.Name()); err != nil {
		return nil, err
	}

	model, exists := f.models[p.Info.ParentType.Name()]
	if !exists {
		return nil, fmt.Errorf("model %s not found", p.Info.ParentType.Name())
	}

	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid input")
	}

	// Строим SQL запрос
	query, args, err := f.buildInsertQuery(model, input)
	if err != nil {
		return nil, err
	}

	// Выполняем запрос
	var result map[string]interface{}
	if err := f.db.Get(&result, query+" RETURNING *", args...); err != nil {
		return nil, err
	}

	return result, nil
}

// buildInsertQuery строит SQL запрос для вставки данных
func (f *Framework) buildInsertQuery(model Model, input map[string]interface{}) (string, []interface{}, error) {
	var columns []string
	var placeholders []string
	var args []interface{}

	for field, value := range input {
		// Проверяем, что поле существует в модели
		var exists bool
		for _, f := range model.Fields {
			if f.Name == field {
				exists = true
				break
			}
		}
		if !exists {
			continue
		}

		columns = append(columns, field)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
		args = append(args, value)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		model.Table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return query, args, nil
}

// resolveRelation обрабатывает связи один-к-одному и один-ко-многим
func (f *Framework) resolveRelation(p graphql.ResolveParams, rel Relation) (interface{}, error) {
	parent, ok := p.Source.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid parent object")
	}

	parentID, ok := parent["id"].(string)
	if !ok {
		return nil, errors.New("parent object has no id")
	}

	// Проверяем права доступа
	if err := f.checkPermissions(p.Context, "read", rel.Target); err != nil {
		return nil, err
	}

	targetModel, exists := f.models[rel.Target]
	if !exists {
		return nil, fmt.Errorf("target model %s not found", rel.Target)
	}

	var query string
	var args []interface{}

	switch rel.Type {
	case "one_to_one":
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", targetModel.Table, rel.ForeignKey)
		args = []interface{}{parentID}
	case "one_to_many":
		query = fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", targetModel.Table, rel.ForeignKey)
		args = []interface{}{parentID}
	}

	var result []map[string]interface{}
	if err := f.db.Select(&result, query, args...); err != nil {
		return nil, err
	}

	if rel.Type == "one_to_one" && len(result) > 0 {
		return result[0], nil
	}

	return result, nil
}

// resolveManyToMany обрабатывает связи многие-ко-многим
func (f *Framework) resolveManyToMany(p graphql.ResolveParams, rel Relation) (interface{}, error) {
	parent, ok := p.Source.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid parent object")
	}

	parentID, ok := parent["id"].(string)
	if !ok {
		return nil, errors.New("parent object has no id")
	}

	// Проверяем права доступа
	if err := f.checkPermissions(p.Context, "read", rel.Target); err != nil {
		return nil, err
	}

	targetModel, exists := f.models[rel.Target]
	if !exists {
		return nil, fmt.Errorf("target model %s not found", rel.Target)
	}

	query := fmt.Sprintf(
		"SELECT t.* FROM %s t JOIN %s j ON t.id = j.%s_id WHERE j.%s_id = $1",
		targetModel.Table,
		rel.JoinTable,
		strings.ToLower(rel.Target),
		strings.ToLower(rel.Name),
	)

	var result []map[string]interface{}
	if err := f.db.Select(&result, query, parentID); err != nil {
		return nil, err
	}

	return result, nil
}

// checkPermissions проверяет права доступа
func (f *Framework) checkPermissions(ctx context.Context, action, model string) error {
	if f.middleware == nil || f.middleware.authFunc == nil {
		return nil
	}

	_, roles, err := f.middleware.authFunc(ctx)
	if err != nil {
		return err
	}

	modelDef, exists := f.models[model]
	if !exists {
		return fmt.Errorf("model %s not found", model)
	}

	var requiredRoles []string
	switch action {
	case "create":
		requiredRoles = modelDef.Permissions.Create
	case "read":
		requiredRoles = modelDef.Permissions.Read
	case "update":
		requiredRoles = modelDef.Permissions.Update
	case "delete":
		requiredRoles = modelDef.Permissions.Delete
	default:
		return fmt.Errorf("unknown action %s", action)
	}

	// Проверяем, есть ли у пользователя хотя бы одна из требуемых ролей
	for _, role := range roles {
		if f.rbac.HasPermission(role, requiredRoles) {
			return nil
		}
	}

	return fmt.Errorf("permission denied for action %s on model %s", action, model)
}

// SetAuthMiddleware устанавливает функцию аутентификации
func (f *Framework) SetAuthMiddleware(authFunc func(ctx context.Context) (string, []string, error)) {
	f.middleware = &Middleware{
		authFunc: authFunc,
	}
}

// Execute выполняет GraphQL запрос
func (f *Framework) Execute(ctx context.Context, query string, variables map[string]interface{}) *graphql.Result {
	return graphql.Do(graphql.Params{
		Context:        ctx,
		Schema:         *f.schema,
		RequestString:  query,
		VariableValues: variables,
	})
}

// HasPermission проверяет, есть ли у роли нужные права
func (r *RBAC) HasPermission(role string, requiredRoles []string) bool {
	permissions, exists := r.roles[role]
	if !exists {
		return false
	}

	for _, required := range requiredRoles {
		if permissions[required] {
			return true
		}
	}

	return false
}

// AddRole добавляет роль с указанными разрешениями
func (r *RBAC) AddRole(role string, permissions []string) {
	if r.roles[role] == nil {
		r.roles[role] = make(map[string]bool)
	}

	for _, perm := range permissions {
		r.roles[role][perm] = true
	}
}
