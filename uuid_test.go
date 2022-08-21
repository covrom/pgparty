package pgparty

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestUUID58_UnmarshalJSON(t *testing.T) {
	b58 := []byte(`{"id":"AnUw9CFxaSCF81BJrbbfiJ"}`)
	uid := UUID58{UUID: uuid.Must(uuid.Parse("4f3beea4-d4a4-4842-bc96-57e818879f03"))}
	type uidStruct struct {
		ID UUID58 `json:"id"`
	}
	var u58 uidStruct
	if err := json.Unmarshal(b58, &u58); err != nil {
		t.Error(err)
		return
	}
	if u58.ID != uid {
		t.Error("not equal after unmarshal")
		return
	}
	bb, err := json.Marshal(u58)
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(bb, b58) {
		t.Error("not equal after marshal")
	}
}
