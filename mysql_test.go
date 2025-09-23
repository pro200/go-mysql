package mysql_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/pro200/go-env"
	"github.com/pro200/go-mysql"
)

/* .config.env
HOST:     string
USERNAME: string
PASSWORD: string
DATABASE: string
*/

/* sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL
);
INSERT INTO users (id, name, email) VALUES
	(1, 'Alice', 'alice@example.com'),
	(3, 'pro200', 'pro200@gmail.com'),
	(4, 'james', 'james@naver.com');
*/

// DB 컬럼과 매핑될 구조체
type User struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func TestMysql(t *testing.T) {
	config, err := env.New()
	if err != nil {
		t.Error(err)
	}

	// MYSQL 연결
	db, err := mysql.New(mysql.Config{
		Host:     config.Get("HOST"),
		Username: config.Get("USERNAME"),
		Password: config.Get("PASSWORD"),
		Database: config.Get("DATABASE"),
		Port:     config.GetInt("PORT"),
	})
	if err != nil {
		log.Fatal("DB 연결 실패:", err)
	}
	defer db.Close()

	// 단일 Row 조회 - 기본 포인터
	var id int
	var name, email string
	err = db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", "alice@example.com", &id, &name, &email)
	if err != nil {
		t.Error("QueryRow 실패:", err)
	}

	fmt.Println("단일 사용자:", id, name, email)

	// 단일 Row 조회 - 구조체 포인터
	var user User
	err = db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", "alice@example.com", &user)
	if err != nil {
		log.Println("QueryRow 실패:", err)
	}

	fmt.Println("단일 사용자(구조체):", user)

	// 값비교
	if name != user.Name {
		t.Error("Wrong result")
	}
}
