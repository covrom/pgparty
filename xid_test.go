package pgparty_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/covrom/pgparty"
)

type myXID struct{}

func (u myXID) XIDPrefix() string { return "myxid_" }

func TestXID(t *testing.T) {
	parseXID, err := pgparty.ParseXID[myXID]("myxid_c900cfj8ejjd3nqe1o30")
	if err != nil {
		t.Errorf("ParseXID error: %s", err)
		return
	}
	if s := parseXID.String(); s != "myxid_c900cfj8ejjd3nqe1o30" {
		t.Errorf("parseXID string is not myxid_c900cfj8ejjd3nqe1o30: %s", s)
		return
	}

	var nilXID pgparty.XID[myXID]
	if s := nilXID.String(); s != "myxid_00000000000000000000" {
		t.Errorf("nilXID string is not myxid_00000000000000000000: %s", s)
		return
	}
	if b, _ := nilXID.MarshalJSON(); !bytes.Equal(b, []byte("null")) {
		t.Errorf("nilXID json is not null")
		return
	}

	if b, _ := parseXID.MarshalJSON(); !bytes.Equal(b, []byte{
		0x22, 0x63, 0x39, 0x30, 0x30, 0x63, 0x66, 0x6a,
		0x38, 0x65, 0x6a, 0x6a, 0x64, 0x33, 0x6e, 0x71, 0x65, 0x31, 0x6f, 0x33, 0x30, 0x22,
	}) {
		t.Errorf("parseXID json is incorrect: %#v", b)
		return
	}

	newXID := pgparty.NewXID[myXID]()
	if s := newXID.String(); s == "myxid_00000000000000000000" {
		t.Errorf("newXID string is myxid_00000000000000000000: %s", s)
		return
	}

	var nilptr *pgparty.XID[myXID]
	rt := reflect.ValueOf(nilptr).Type()
	if s := pgparty.SQLType(rt, 0, 0); s != "CHAR(20)" {
		t.Errorf("newXID postgres type is not char(20): %s", s)
		return
	}

	rt = reflect.ValueOf(newXID).Type()
	if s := pgparty.SQLType(rt, 0, 0); s != "CHAR(20)" {
		t.Errorf("newXID postgres type is not char(20): %s", s)
		return
	}

	if err := (&newXID).Scan([]byte{
		0x22, 0x63, 0x39, 0x30, 0x30, 0x63, 0x66, 0x6a,
		0x38, 0x65, 0x6a, 0x6a, 0x64, 0x33, 0x6e, 0x71, 0x65, 0x31, 0x6f, 0x33, 0x30, 0x22,
	}); err == nil {
		t.Errorf("Scan error: %s", err)
		return
	}

	if err := (&newXID).UnmarshalJSON([]byte(`"c900cfj8ejjd3nqe1o30"`)); err != nil {
		t.Errorf("Scan error: %s", err)
		return
	}

	if err := (&newXID).Scan([]byte("c900cfj8ejjd3nqe1o30")); err != nil {
		t.Errorf("Scan error: %s", err)
		return
	}

	if newXID != parseXID {
		t.Errorf("newXID!=parseXID")
		return
	}

	if err := (&newXID).Scan("c900cfj8ejjd3nqe1o30"); err != nil {
		t.Errorf("Scan error: %s", err)
		return
	}

	if newXID != parseXID {
		t.Errorf("newXID!=parseXID")
		return
	}

	v, err := newXID.Value()
	if err != nil {
		t.Errorf("Value error: %s", err)
		return
	}
	if vv, ok := v.(string); !ok || vv != "c900cfj8ejjd3nqe1o30" {
		t.Errorf("vv != c900cfj8ejjd3nqe1o30, but %#v", v)
		return
	}

	if v, err := nilXID.Value(); err != nil || v != nil {
		t.Errorf("nilXID.Value not nil, but %#v", v)
		return
	}

	var jxid pgparty.XIDJsonTyped[myXID]
	if err := (&jxid).UnmarshalJSON([]byte(`"c900cfj8ejjd3nqe1o30"`)); err != nil {
		t.Errorf("Scan error: %s", err)
		return
	}
	if err := (&jxid).UnmarshalJSON([]byte(`"myxid_c900cfj8ejjd3nqe1o30"`)); err != nil {
		t.Errorf("Scan error: %s", err)
		return
	}
	jsid, err := json.Marshal(jxid)
	if err != nil {
		t.Errorf("json.Marshal(jxid) error: %s", err)
		return
	}
	if string(jsid) != `"myxid_c900cfj8ejjd3nqe1o30"` {
		t.Errorf(`jsid error: %s != "myxid_c900cfj8ejjd3nqe1o30"`, string(jsid))
		return
	}
}
