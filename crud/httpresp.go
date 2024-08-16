package crud

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/covrom/pgparty"
	"github.com/go-chi/render"
)

type DatabaseListQuerier[T pgparty.Modeller] interface {
	ResponseList() ([]pgparty.JsonViewer[T], error)
}

func ResponseList[T pgparty.Modeller, Q DatabaseListQuerier[T]](w http.ResponseWriter, r *http.Request, q Q) {
	resp, err := q.ResponseList()
	if err != nil {
		log.Println(err)
		render.Render(w, r, ErrRender(err))
		return
	}
	fmt.Fprint(w, "[")
	defer fmt.Fprint(w, "]")

	enc := json.NewEncoder(w)

	for i, v := range resp {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		jsv := v.JsonView()
		enc.Encode(jsv)
	}
}
