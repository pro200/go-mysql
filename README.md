# MySQL Wrapper for Go

Go 언어에서 [`database/sql`](https://pkg.go.dev/database/sql) 과 [`go-sql-driver/mysql`](https://github.com/go-sql-driver/mysql) 를 기반으로 만든 **간단한 MySQL 래퍼 패키지**입니다.  
구조체 태그(`db:"column"`)를 이용하여 쿼리 결과를 자동 매핑할 수 있습니다.

---

## ✨ 특징
- 멀티 DB 연결 지원 (`Databases["name"]`)
- `db` 태그 기반 구조체 매핑
- 단일 행(`QueryRow`) / 다중 행(`Query`) 조회 지원
- `Exec`, `ExecOne` (LIMIT 1 자동 추가) 지원
- 기본값 자동 처리 (포트, 프로토콜 등)
- 스레드 안전한 DB 핸들 관리

---

## 📦 설치
```bash
go get github.com/yourname/mysql
```

## ⚙️ 설정 및 사용 예제

테이블 준비
```sql
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL
);
```

구조체 정의
```go
type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}
```

연결 및 CRUD 예제
```go
package main

import (
    "fmt"
    "log"

    "github.com/pro200/go-mysql"
)

func main() {
    // DB 연결
    db, err := mysql.New(mysql.Config{
        Host:     "127.0.0.1",
        Username: "root",
        Password: "1234",
        Database: "testdb",
    })
    if err != nil {
        log.Fatal("DB 연결 실패:", err)
    }
    defer db.Close()

    // INSERT
    _, err = db.Exec("INSERT INTO users(name, email) VALUES(?, ?)", "Alice", "alice@example.com")
    if err != nil {
        log.Println("Insert 실패:", err)
    }

    // 단일 Row 조회
    var u User
    err = db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", "alice@example.com", &u)
    if err != nil {
        log.Println("QueryRow 실패:", err)
    } else {
        fmt.Printf("단일 사용자: %+v\n", u)
    }

    // 다중 Row 조회
    var users []User
    err = db.Query("SELECT id, name, email FROM users", &users)
    if err != nil {
        log.Println("Query 실패:", err)
    } else {
        fmt.Println("사용자 목록:")
        for _, user := range users {
            fmt.Printf(" - %+v\n", user)
        }
    }

    // UPDATE (한 건만 수정)
    _, err = db.ExecOne("UPDATE users SET name = ? WHERE email = ?", "Alice Updated", "alice@example.com")
    if err != nil {
        log.Println("Update 실패:", err)
    }

    // DELETE
    _, err = db.Exec("DELETE FROM users WHERE email = ?", "alice@example.com")
    if err != nil {
        log.Println("Delete 실패:", err)
    }
}
```
---
## 📚 API
New(config Config) (*Database, error)
- 새로운 DB 연결 생성 및 등록
- 기본값: Name="main", Port=3306, Protocol=tcp, ConnMaxHour=1, MaxOpenConns=128, MaxIdleConns=10,

GetDatabase(name ...string) (*Database, error)
- 등록된 DB 핸들 가져오기 (기본: "main")

QueryRow(query string, args ...any) error
- 단일 행 조회
- 마지막 인자는 구조체 포인터 or 기본 포인터
- db 태그 기반 매핑

Query(query string, args ...any) error
- 다중 행 조회
- 마지막 인자는 &[]Struct 형태여야 함

Exec(query string, args ...any) (sql.Result, error)
- 일반 INSERT, UPDATE, DELETE

ExecOne(query string, args ...any) (sql.Result, error)
- LIMIT 1이 자동 추가된 실행

Close() error
- DB 연결 종료

## ⚠️ 주의사항
- 구조체 매핑은 db:"컬럼명" 태그를 기준으로 동작합니다.
- 태그가 없으면 필드명을 소문자로 변환하여 매핑합니다.
- Databases 전역 맵을 사용할 때는 동일한 이름 중복 등록을 피하세요.