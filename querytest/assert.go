package querytest

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// AssertCount executes a SQL statement with the form 'SELECT COUNT(*) FROM ...' and
// fails the test with a descriptive error message if the result value returned is not
// equal to wantCount
func AssertCount(t *testing.T, tx *sql.Tx, wantCount int, query string, args ...any) {
	row := tx.QueryRow(query, args...)

	var count int
	err := row.Scan(&count)
	if err == nil && count != wantCount {
		err = fmt.Errorf("expected count of %d; got %d", wantCount, count)
	}

	if err != nil {
		t.Logf("With query:")
		for _, line := range strings.Split(query, "\n") {
			t.Logf("  %s", line)
		}
		if len(args) > 0 {
			t.Logf("With args:")
			for i, value := range args {
				t.Logf(" $%d: %v", i+1, value)
			}
		}
		t.Fatalf(err.Error())
	}
}

// AssertNumRowsChanged inspects a sql.Result to verify that the expected number of rows
// were changed by the operation. Useful for validating queries that update records
// conditionally, etc.
func AssertNumRowsChanged(t *testing.T, res sql.Result, wantNumRows int) {
	numRows, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("failed to get number of rows changed: %v", err)
	}
	if numRows != int64(wantNumRows) {
		t.Fatalf("expected operation to change %d rows; instead it changed %d", wantNumRows, numRows)
	}
}
