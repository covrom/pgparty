package crud_test

import (
	"context"
	"net/http"

	"github.com/covrom/pgparty"
	"github.com/covrom/pgparty/crud"
	"github.com/jmoiron/sqlx"
)

type Board struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Disabled bool   `json:"hided"`
}

func (m Board) DatabaseName() string {
	return "boards"
}
func (Board) TypeName() pgparty.TypeName {
	return pgparty.StructModel[Board]{}.TypeName()
}
func (Board) Fields() []pgparty.FieldDescription {
	return pgparty.StructModel[Board]{}.Fields()
}

var _ crud.DatabaseListQuerier[Board] = &GetBoardsOrderedQuery{}

// boards ordered by position,id
type GetBoardsOrderedQuery struct {
	Ctx context.Context
	eng *pgparty.ModelQueryEngine[Board, *GetBoardsOrderedQuery]
}

func (q *GetBoardsOrderedQuery) QueryRows() (*sqlx.Rows, error) {
	return pgparty.Query(q.Ctx,
		`SELECT :ID FROM &Board WHERE NOT :Disabled ORDER BY :Position, :ID`,
	)
}

func (q *GetBoardsOrderedQuery) ResponseList() ([]pgparty.JsonViewer[Board], error) {
	if q.eng == nil {
		q.eng = &pgparty.ModelQueryEngine[Board, *GetBoardsOrderedQuery]{}
	}
	return q.eng.ResponseList(q)
}

func ExampleGetBoardsOrderedQuery() {
	_ = func(w http.ResponseWriter, r *http.Request) {
		// crud.SelectQ(r.Context(), `SELECT :ID FROM &Board WHERE NOT :Disabled ORDER BY :Position, :ID`, string("")).
		// 	Do().ServeHTTP(w, r)

		q := &GetBoardsOrderedQuery{
			Ctx: r.Context(),
		}
		crud.ResponseList[Board](w, r, q)
	}
}
