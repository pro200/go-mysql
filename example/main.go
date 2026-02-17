package main

import (
	"errors"
	"fmt"

	"github.com/pro200/go-config"
	"github.com/pro200/go-mysql"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	// MYSQL 연결
	db, err := mysql.New(mysql.Config{
		Host:     cfg.Get("HOST"),
		Username: cfg.Get("USERNAME"),
		Password: cfg.Get("PASSWORD"),
		Database: cfg.Get("DATABASE"),
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
