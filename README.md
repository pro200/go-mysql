```go
package main

import (
	"github.com/pro200/go-mysql"
	"github.com/pro200/go-env"
	"fmt"
)

type User struct {
	Id    string
	Email string
}

func main() {
	config, err := env.New()
	if err != nil {
		t.Error(err)
	}
	
	mysql.Init(mysql.Config{
		Host:         config.Get("HOST"),
		Username:     config.Get("USERNAME"),
		Password:     config.Get("PASSWORD"),
		Database:     config.Get("DATABASE"),
		MaxIdleConns: 10,
	})
	defer mysql.Close()

	var user User
	if err := mysql.QueryRow("SELECT `mbId`, `mbEmail` FROM `member` WHERE mbNo=?", 1, &user); err != nil {
		panic(err)
	}
	fmt.Println(user.Id, user.Email)

	var id, email string
	if err := mysql.QueryRow("SELECT `mbId`, `mbEmail` FROM `member` WHERE mbNo=?", 30, &id, &email); err != nil {
		panic(err)
	}
	fmt.Println(id, email)

	var users []User
	if err := mysql.Query("SELECT `mbId`, `mbEmail` FROM `member` limit ?", 3, &users); err != nil {
		panic(err)
	}

	for _, user := range users {
		fmt.Println(user.Id, user.Email)
	}
}
```