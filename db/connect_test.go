package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FormatConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		dbname   string
		user     string
		password string
		sslmode  string
		want     string
	}{
		{
			"normal usage",
			"localhost",
			5432,
			"somedb",
			"someuser",
			"password",
			"",
			"postgres://someuser:password@localhost:5432/somedb",
		},
		{
			"sslmode is appended if non-empty",
			"my-postgres-server.biz",
			5444,
			"mydb",
			"myuser",
			"mypassword",
			"disable",
			"postgres://myuser:mypassword@my-postgres-server.biz:5444/mydb?sslmode=disable",
		},
		{
			"password can contain special characters, is url-encoded",
			"localhost",
			5432,
			"somedb",
			"someuser",
			"pass:@/word",
			"",
			"postgres://someuser:pass%3A%40%2Fword@localhost:5432/somedb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatConnectionString(tt.host, tt.port, tt.dbname, tt.user, tt.password, tt.sslmode)
			assert.Equal(t, tt.want, got)
		})
	}
}
