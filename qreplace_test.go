package pgparty

import (
	"fmt"
	"strings"
	"testing"
)

func TestScanWords(t *testing.T) {

	query := `Update a.&Model1 SET :Model1.ID-1+2*3/4=? WHERE :ID=? --:COMMENT1
		FROM SELECT :ID -3 - 5,:Model1.*,:Model2.ID,:Model1.Name,alias.:Model1.ID,:Model2_id 
		-- comment with &SUPERMODEL
FROM &CURRSCHEMA.&Model1=? LEFT JOIN &Model2.Many2ManyField-1 ON :Model1.ID=:Model2.FID	
			AND :Model1.ID ?& array["a:F'X'&N", 'b&c   ?:L']`
	words := scanParamsAndQueries(query)
	sb := &strings.Builder{}
	for _, w := range words {
		fmt.Fprintf(sb, "%q %q\n", w.query, w.param)
	}

	if sb.String() != `"Update " ""
"a.&Model1 " "&Model1"
"SET " ""
":Model1.ID-" ":Model1.ID"
"1+" ""
"2*3/" ""
"4=" ""
"? WHERE " ""
":ID=" ":ID"
"? " ""
" FROM " ""
"SELECT " ""
":ID " ":ID"
"-3 " ""
"- 5," ""
":Model1.*," ":Model1.*"
":Model2.ID," ":Model2.ID"
":Model1.Name," ":Model1.Name"
"alias.:Model1.ID," ":Model1.ID"
":Model2_id " ":Model2_id"
" " ""
"FROM " ""
"&CURRSCHEMA.&Model1=" "&CURRSCHEMA.&Model1"
"? LEFT " ""
"JOIN " ""
"&Model2.Many2ManyField-" "&Model2.Many2ManyField"
"1 " ""
"ON " ""
":Model1.ID=" ":Model1.ID"
":Model2.FID " ":Model2.FID"
" AND " ""
":Model1.ID " ":Model1.ID"
"?& " ""
"array[" ""
"\"a:F'X'&N\", 'b&c   ?:L']" ""
` {
		t.Error(sb.String())
	}

}
