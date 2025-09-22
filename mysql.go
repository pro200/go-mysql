package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Name         string // database 이름
	Host         string
	Port         int
	Protocol     string
	Username     string
	Password     string
	Database     string
	MaxIdleConns int
}

type Database struct {
	client *sql.DB
}

var Databases = make(map[string]*Database)

// New 생성자
func New(config Config) *Database {
	if config.Name == "" {
		config.Name = "main"
	}
	if config.Port == 0 {
		config.Port = 3306
	}
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}

	option := fmt.Sprintf("%s:%s@%s(%s)/%s", config.Username, config.Password, config.Protocol, config.Host, config.Database)
	db, err := sql.Open("mysql", option)
	if err != nil {
		panic("Mysql connection error. Check the env 'DB_HOST'\n       " + err.Error())
	}

	// db.SetConnMaxLifetime(0) // zero means unlimited
	db.SetMaxOpenConns(128)                 // <= 0 means unlimited
	db.SetMaxIdleConns(config.MaxIdleConns) // zero means defaultMaxIdleConns = 2; negative means 0

	Databases[config.Name] = &Database{client: db}
	return Databases[config.Name]
}

func GetDatabase(name ...string) (*Database, error) {
	if len(Databases) == 0 {
		return nil, errors.New("no databases available")
	}

	dbName := "main"
	if len(name) > 0 {
		dbName = name[0]
	}
	db, ok := Databases[dbName]
	if !ok {
		return nil, fmt.Errorf("database %s not found", dbName)
	}
	return db, nil
}

// args: values, dests로 분류되며 일반변수와 포인터로 기준은 나눈다. 구조체로 받을경우는 한개의 dest를 사용한다.
// 구조체 필드 순서가 쿼리 결과 순서와 일치해야 하며 구조체의 필드가 많을 경우 나머지 필드는 무시된다.
//
// ex) "pro200", "23", &id, &email -> ["pro200", "23"]는 쿼리 값이고, [&id, &email] 데이터를 매핑할 dests이다.
// ex) "pro200", "23", &user -> ["pro200", "23"]는 쿼리 값이고, &user는 구조체이거나 일반변수일 수 있다.
func (db *Database) QueryRow(query string, args ...any) error {
	var values, dests []interface{}

	// values, dests를 분리
	for i, r := range args {
		val := reflect.ValueOf(r)
		if val.Kind() == reflect.Ptr {
			values, dests = args[:i], args[i:]
			break
		}
	}
	// dests는 모두 포인터여야 한다
	for _, r := range dests {
		val := reflect.ValueOf(r)
		if val.Kind() != reflect.Ptr {
			return fmt.Errorf("dests must be a pointer")
		}
	}

	/*
	 * dests의 마지막 값이 구조체가 아니면 QueryRow를 사용한다
	 */
	if reflect.ValueOf(dests[len(dests)-1]).Elem().Kind() != reflect.Struct {
		err := db.client.QueryRow(query, values...).Scan(dests...)
		if err != nil {
			return err
		}
		return nil
	}

	/*
	 * dest가 구조체일 경우 Query를 사용한다
	 */
	dest := dests[len(dests)-1]
	destVal := reflect.ValueOf(dest)
	// reflect.Ptr 이건 포인터
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to a struct")
	}

	elem := destVal.Elem()

	// query에 LIMIT이 없으면 추가
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT 1", query)
	}

	// Query를 사용해 sql.Rows 객체 가져오기
	rows, err := db.client.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 첫 번째 행만 처리
	if !rows.Next() {
		return sql.ErrNoRows
	}

	// 열 개수를 확인하여 필드 포인터 배열 생성
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	fieldPointers := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		if i < elem.NumField() {
			fieldPointers[i] = elem.Field(i).Addr().Interface()
		} else {
			// 열의 개수가 구조체 필드 수를 초과하는 경우 dummy 변수를 사용
			var dummy interface{}
			fieldPointers[i] = &dummy
		}
	}

	// Scan으로 각 필드에 값을 채워넣음
	if err := rows.Scan(fieldPointers...); err != nil {
		return err
	}

	return nil
}

// args: values, dest로 분류되며 마지막 값은 반드시 포인터 구조체 슬라이스 이다.
// 구조체 필드 순서가 쿼리 결과 순서와 일치해야 하며 구조체의 필드가 많을 경우 나머지 필드는 무시된다.
//
// ex) "pro200", "23", &user -> ["pro200", "23"]는 쿼리 값이고, &user는 구조체 슬라이스이다
func (db *Database) Query(query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("need more args")
	}
	// args의 마지막 값은 dest이며, 나머지는 sql의 값이다
	dest, values := args[len(args)-1], args[:len(args)-1]

	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceVal := destVal.Elem()
	elemType := sliceVal.Type().Elem()

	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("elements of dest must be structs")
	}

	rows, err := db.client.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	for rows.Next() {
		elem := reflect.New(elemType).Elem()

		// 각 필드에 대한 포인터 생성
		fieldPointers := make([]interface{}, len(cols))
		for i := 0; i < len(cols); i++ {
			if i < elem.NumField() {
				fieldPointers[i] = elem.Field(i).Addr().Interface()
			} else {
				// 필드 수가 더 적은 경우 추가 열을 무시하기 위해 dummy 변수를 사용
				var dummy interface{}
				fieldPointers[i] = &dummy
			}
		}

		// Scan으로 각 필드에 값을 채워넣음
		if err := rows.Scan(fieldPointers...); err != nil {
			return err
		}

		sliceVal.Set(reflect.Append(sliceVal, elem))
	}

	return nil
}

func (db *Database) Exec(query string, args ...any) (sql.Result, error) {
	return db.client.Exec(query, args...)
}

func (db *Database) ExecOne(query string, args ...any) (sql.Result, error) {
	// query에 LIMIT이 없으면 추가
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT 1", query)
	}

	return db.client.Exec(query, args...)
}
