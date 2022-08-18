package pgparty

import "testing"

func TestIdentifyPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(IdentifyPanic())
		}
	}()
	panic("ddd")
}
