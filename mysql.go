package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/cloudfoundry-community/go-cfenv"
)

const (
	mySQLKey  = "mySQL"
	mySQLName = "MySQL"

	mySQLTestBinding       = "Read service binding"
	mySQLErrorBinding      = "No MySQL hostname found in VCAP_SERVICES"
	mySQLTestConnection    = "Open connection"
	mySQLTestPrepareCreate = "Prepare create table"
	mySQLTestCreate        = "Create table"
	mySQLTestPrepareInsert = "Prepare insert record"
	mySQLTestInsert        = "Insert record"
	mySQLTestSelect        = "Select records"
	mySQLTestPrepareDelete = "Prepare delete record"
	mySQLTestDelete        = "Delete record"
)

type mySQLTest struct {
	hostname string
	port     float64
	dbname   string
	username string
	password string
}

func mySQLTestNew(env *cfenv.App) SmokeTest {
	// TODO: replace with searching on tag basis, possibly resulting in multiple returns in case of multiple matches.
	mySQLServices, err := env.Services.WithLabel("p.mySQL")
	if err != nil {
		return &mySQLTest{"", 0.0, "", "", ""}
	}

	creds := mySQLServices[0].Credentials
	return &mySQLTest{
		hostname: creds["hostname"].(string),
		port:     creds["port"].(float64),
		dbname:   creds["name"].(string),
		username: creds["username"].(string),
		password: creds["password"].(string),
	}

}

func (m *mySQLTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	// Check service binding.
	fmt.Println("found mySQL hostname:" + m.hostname)
	if m.hostname == "" {
		fmt.Println("no mySQL uri found")
		results = append(results, SmokeTestResult{Name: mySQLTestBinding, Result: false, Error: mySQLErrorBinding})
		return OverallResult(mySQLKey, mySQLName, results)
	}
	results = append(results, SmokeTestResult{Name: mySQLTestBinding, Result: true})

	// Open connection.
	openConnection := func() (interface{}, error) {
		return sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%.f)/%v?readTimeout=30s&writeTimeout=30s&timeout=30s", m.username, m.password, m.hostname, m.port, m.dbname))
	}
	obj, success := RunTestPart(openConnection, mySQLTestConnection, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}
	db := obj.(*sql.DB)
	defer db.Close()

	// Prepare create table.
	prepareCreateTable := func() (interface{}, error) {
		return db.Prepare("CREATE TABLE IF NOT EXISTS deepthought(theanswertoeverything INT)")
	}
	obj, success = RunTestPart(prepareCreateTable, mySQLTestPrepareCreate, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}
	createTableStmt := obj.(*sql.Stmt)
	defer createTableStmt.Close()

	// Create table.
	createTable := func() (interface{}, error) {
		return createTableStmt.Exec()
	}
	_, success = RunTestPart(createTable, mySQLTestCreate, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}

	// Prepare insert.
	prepareInsert := func() (interface{}, error) {
		return db.Prepare("INSERT INTO deepthought(theanswertoeverything) VALUES(?)")
	}
	obj, success = RunTestPart(prepareInsert, mySQLTestPrepareInsert, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}
	insertStmt := obj.(*sql.Stmt)
	defer insertStmt.Close()

	// Insert.
	insert := func() (interface{}, error) {
		return insertStmt.Exec(42)
	}
	_, success = RunTestPart(insert, mySQLTestInsert, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}

	// Select.
	query := func() (interface{}, error) {
		return db.Query("SELECT * FROM deepthought WHERE theanswertoeverything = 42")
	}
	_, success = RunTestPart(query, mySQLTestSelect, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}

	// Prepare delete.
	prepareDelete := func() (interface{}, error) {
		return db.Prepare("DELETE FROM deepthought WHERE theanswertoeverything = ?")
	}
	obj, success = RunTestPart(prepareDelete, mySQLTestPrepareDelete, &results)
	if !success {
		return OverallResult(mySQLKey, mySQLName, results)
	}
	deleteStmt := obj.(*sql.Stmt)
	defer deleteStmt.Close()

	// Delete.
	delete := func() (interface{}, error) {
		return deleteStmt.Exec("42")
	}
	_, _ = RunTestPart(delete, mySQLTestDelete, &results)

	// Determine overall result and return.
	return OverallResult(mySQLKey, mySQLName, results)
}
