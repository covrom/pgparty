package pgparty_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/covrom/pgparty"
)

type Test struct {
	ID          pgparty.UUIDv4 `json:"id"`
	Name        string         `json:"name"`
	Description pgparty.String `json:"desc"`
	Subs        []SubTest      `json:"subs"`
}

func (Test) StoreName() string { return "test" }

type SubTest struct {
	ID   pgparty.UUIDv4 `json:"id"`
	Name string         `json:"name"`
}

func TestJsonViewUnmarshalJSON(t *testing.T) {
	el := Test{
		ID:          pgparty.UUIDNewV4(),
		Name:        "simple name",
		Description: "second name",
		Subs: []SubTest{
			{
				ID:   pgparty.UUIDNewV4(),
				Name: "sub1",
			},
			{
				ID:   pgparty.UUIDNewV4(),
				Name: "sub2",
			},
		},
	}
	jv, err := pgparty.NewJsonView[Test]()
	if err != nil {
		t.Error(err)
		return
	}
	b, err := json.Marshal(el)
	if err != nil {
		t.Error(err)
		return
	}
	if err := json.Unmarshal(b, jv); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(jv.V, el) {
		t.Logf("%+v", jv.V)
		t.Error()
	}
}
