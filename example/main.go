package main

import (
	"errors"
	"fmt"

	"github.com/pro200/go-env"
	"github.com/pro200/go-mysql"
)

func main() {
	config, err := env.New()
	if err != nil {
		panic(err)
	}

	// MYSQL 연결
	db, err := mysql.NewDatabase(mysql.Config{
		Host:     config.Get("HOST"),
		Username: config.Get("USERNAME"),
		Password: config.Get("PASSWORD"),
		Database: config.Get("DATABASE"),
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var member struct {
		Id    string `db:"mbId"`
		Email string `db:"mbEmail"`
	}
	err = db.QueryRow(
		"SELECT `mbId`, `mbEmail` " +
		"FROM `member` " +
		"WHERE `mbNo`=?",
		1, &member)
	if err != nil {
		panic(err)
	}

	fmt.Println(member)

	var members []struct {
		Id    string `db:"mbId"`
		Email string `db:"mbEmail"`
	}

	err = db.Query("SELECT `mbId`, `mbEmail` FROM `member` WHERE `mbNo`>=1 limit 3", &members)
	if err != nil {
		if errors.Is(err, mysql.ErrNoResult) {
			fmt.Println("No result")
		}
		panic(err)
	}

	fmt.Println(members)
}
