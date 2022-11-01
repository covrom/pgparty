package pgparty

import (
	"reflect"
	"testing"
)

type testView struct {
}

func (testView) StoreName() string {
	return "testView"
}

func (testView) ViewQuery() string {
	return "select 1"
}

type testView2 struct {
	testView
}

func (testView2) MaterializedView() {
}

type testViewNone struct {
}

func (testViewNone) StoreName() string {
	return "testViewNone"
}

func Test_viewAttrs(t *testing.T) {
	type args struct {
		typ reflect.Type
	}
	tests := []struct {
		name               string
		args               args
		wantIsView         bool
		wantIsMaterialized bool
		wantViewQuery      string
	}{
		{
			name: "1",
			args: args{
				reflect.TypeOf(testView{}),
			},
			wantIsView:         true,
			wantIsMaterialized: false,
			wantViewQuery:      "select 1",
		},
		{
			name: "2",
			args: args{
				reflect.TypeOf(testView2{}),
			},
			wantIsView:         true,
			wantIsMaterialized: true,
			wantViewQuery:      "select 1",
		},
		{
			name: "none",
			args: args{
				reflect.TypeOf(testViewNone{}),
			},
			wantIsView:         false,
			wantIsMaterialized: false,
			wantViewQuery:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsView, gotIsMaterialized, gotViewQuery := viewAttrs(tt.args.typ)
			if gotIsView != tt.wantIsView {
				t.Errorf("viewAttrs() gotIsView = %v, want %v", gotIsView, tt.wantIsView)
			}
			if gotIsMaterialized != tt.wantIsMaterialized {
				t.Errorf("viewAttrs() gotIsMaterialized = %v, want %v", gotIsMaterialized, tt.wantIsMaterialized)
			}
			if gotViewQuery != tt.wantViewQuery {
				t.Errorf("viewAttrs() gotViewQuery = %v, want %v", gotViewQuery, tt.wantViewQuery)
			}
		})
	}
}
