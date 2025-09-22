package mysql_test

import (
	"fmt"
	"github.com/pro200/go-env"
	"github.com/pro200/go-mysql"
	"testing"
)

/* .config.env
HOST:     string
USERNAME: string
PASSWORD: string
DATABASE: string
*/

func TestMysql(t *testing.T) {
	config, err := env.New()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(config)

}
