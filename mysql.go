package mysql

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host         string
	Port         int    // (기본값: 3306)
	Protocol     string // (기본값: "tcp")
	Username     string
	Password     string
	Database     string
	ConnMaxHour  int // < 0 means unlimited (기본값: 1)
	MaxOpenConns int // < 0 means unlimited (기본값: 128)
	MaxIdleConns int // <= 0 means 10 (기본값: 10)
}

type Database struct {
	client *sql.DB
}

// New 생성자
func NewDatabase(config Config) (*Database, error) {
	if config.Port == 0 {
		config.Port = 3306
	}
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
	if config.ConnMaxHour < 0 {
		config.ConnMaxHour = 0
	} else if config.ConnMaxHour == 0 {
		config.ConnMaxHour = 1
	}
	if config.MaxOpenConns < 0 {
		config.MaxOpenConns = 0
	} else if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 128
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 10
	}

	option := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?parseTime=true",
		config.Username, config.Password, config.Protocol, config.Host, config.Port, config.Database)

	db, err := sql.Open("mysql", option)
	if err != nil {
		return nil, fmt.Errorf("mysql connection error: %w", err)
	}

	// 연결 검증
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping error: %w", err)
	}

	db.SetConnMaxLifetime(time.Hour * time.Duration(config.ConnMaxHour)) // 0 means unlimited
	db.SetMaxOpenConns(128)                                              // <= 0 means unlimited
	db.SetMaxIdleConns(config.MaxIdleConns)                              // <= 0 means defaultMaxIdleConns = 2

	return &Database{client: db}, nil
}

// map column name -> struct field index
func mapColumnsToFields(cols []string, elem reflect.Value) ([]interface{}, error) {
	fieldPointers := make([]interface{}, len(cols))
	fieldMap := make(map[string]int)

	// 구조체의 db 태그 기반 매핑
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Type().Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			fieldMap[strings.ToLower(tag)] = i
		} else {
			fieldMap[strings.ToLower(field.Name)] = i
		}
	}

	for i, col := range cols {
		if idx, ok := fieldMap[strings.ToLower(col)]; ok {
			fieldPointers[i] = elem.Field(idx).Addr().Interface()
		} else {
			var dummy interface{}
			fieldPointers[i] = &dummy
		}
	}

	return fieldPointers, nil
}

// 단일 Row 조회 → dest는 반드시 포인터(struct or 기본 타입)
func (db *Database) QueryRow(query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("no destination provided")
	}

	// values, dests를 분리
	var values, dests []interface{}
	for i, r := range args {
		if reflect.TypeOf(r).Kind() == reflect.Ptr {
			values, dests = args[:i], args[i:]
			break
		}
	}

	if len(dests) == 0 {
		return fmt.Errorf("no destination pointer provided")
	}

	// 마지막 dest
	dest := dests[len(dests)-1]
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	// 기본 포인터 매핑
	if destVal.Elem().Kind() != reflect.Struct {
		return db.client.QueryRow(query, values...).Scan(dests...)
	}

	// 구조체 매핑
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT 1", query)
	}

	rows, err := db.client.Query(query, values...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	elem := destVal.Elem()
	fieldPointers, err := mapColumnsToFields(cols, elem)
	if err != nil {
		return err
	}

	if err := rows.Scan(fieldPointers...); err != nil {
		return err
	}
	return nil

}

// 다중 Row 조회 → dest는 반드시 *[]Struct 포인터
func (db *Database) Query(query string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("need more args")
	}
	dest := args[len(args)-1]
	values := args[:len(args)-1]

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

		fieldPointers, err := mapColumnsToFields(cols, elem)
		if err != nil {
			return err
		}

		if err := rows.Scan(fieldPointers...); err != nil {
			return err
		}

		sliceVal.Set(reflect.Append(sliceVal, elem))
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (db *Database) Exec(query string, args ...any) (sql.Result, error) {
	return db.client.Exec(query, args...)
}

func (db *Database) ExecOne(query string, args ...any) (sql.Result, error) {
	if !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT 1", query)
	}
	return db.client.Exec(query, args...)
}

func (db *Database) Close() error {
	return db.client.Close()
}
