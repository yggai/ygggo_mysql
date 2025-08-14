package ygggo_mysql

import (
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

func TestClassify_MySQLErrorCodes(t *testing.T) {
	cases := []struct{
		code uint16
		want ErrorClass
		name string
	}{
		{1213, ErrClassRetryable, "deadlock"},              // ER_LOCK_DEADLOCK
		{1205, ErrClassRetryable, "lock_wait_timeout"},    // ER_LOCK_WAIT_TIMEOUT
		{1290, ErrClassReadonly,  "read_only_mode"},       // ER_OPTION_PREVENTS_STATEMENT (read-only)
		{1062, ErrClassConflict,  "duplicate_entry"},      // ER_DUP_ENTRY
		{1022, ErrClassConflict,  "duplicate_key"},        // ER_DUP_KEY
		{1048, ErrClassConstraint, "not_null"},            // ER_BAD_NULL_ERROR
		{1452, ErrClassConstraint, "fk_no_referenced"},    // ER_NO_REFERENCED_ROW_2
		{1451, ErrClassConstraint, "fk_row_referenced"},   // ER_ROW_IS_REFERENCED_2
		{3819, ErrClassConstraint, "check_violation"},     // ER_CHECK_CONSTRAINT_VIOLATED
		{9999, ErrClassUnknown,   "unknown"},
	}
	for _, tc := range cases {
		if got := Classify(&mysql.MySQLError{Number: tc.code}); got != tc.want {
			t.Fatalf("%s: classify(%d)=%v want %v", tc.name, tc.code, got, tc.want)
		}
	}
}

func TestIsRetryable_IncludesDeadlockTimeoutReadonly(t *testing.T) {
	codes := []uint16{1213, 1205, 1290}
	for _, c := range codes {
		if !isRetryable(adapt(&mysql.MySQLError{Number: c})) {
			t.Fatalf("code %d expected retryable", c)
		}
	}
	if isRetryable(adapt(&mysql.MySQLError{Number: 1062})) { // duplicate should not be retryable
		t.Fatalf("duplicate should not be retryable")
	}
}

