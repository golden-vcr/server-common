package db

import (
	"fmt"
	"net/url"
)

// FormatConnectionString formats the provided database connection details into a
// 'postgres://' URI that can be used to connect to that database, e.g. via sql.Open
func FormatConnectionString(host string, port int, dbname, user, password, sslmode string) string {
	urlencodedPassword := url.QueryEscape(password)
	s := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, urlencodedPassword, host, port, dbname)
	if sslmode != "" {
		s += fmt.Sprintf("?sslmode=%s", sslmode)
	}
	return s
}
