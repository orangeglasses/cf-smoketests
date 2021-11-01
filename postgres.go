package main

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/cloudfoundry-community/go-cfenv"
)

const (
	postgresTestBinding       = "Read service binding"
	postgresErrorBinding      = "No Postgres hostname found in VCAP_SERVICES"
	postgresTestConnection    = "Open connection"
	postgresTestPrepareCreate = "Prepare create table"
	postgresTestCreate        = "Create table"
	postgresTestPrepareInsert = "Prepare insert record"
	postgresTestInsert        = "Insert record"
	postgresTestSelect        = "Select records"
	postgresTestPrepareDelete = "Prepare delete record"
	postgresTestDelete        = "Delete record"

	postgresTestInitialize  = "Initialize"
	postgresErrorInitialize = "Service %v not or incorrectly configured in VCAP_SERVICES"
)

type postgresTest struct {
	host string
	uri  string
	init bool
	key  string
	name string
}

func postgresTestNew(env *cfenv.App, serviceName, friendlyName string) SmokeTest {
	postgresServices, err := env.Services.WithLabel(serviceName)
        if err != nil {
                fmt.Println("Postgres service not bound to smoketest app.")
                return nil
        }

	creds := postgresServices[0].Credentials
	return &postgresTest{
		host: creds["host"].(string),
		uri:  creds["uri"].(string),
		init: true,
		key:  serviceName,
		name: friendlyName,
	}
}

func (m *postgresTest) run() SmokeTestResult {
	results := make([]SmokeTestResult, 0)

	if !m.init {
		results = append(results, SmokeTestResult{Name: postgresTestInitialize, Result: false, Error: fmt.Sprintf(postgresErrorInitialize, m.key)})
		return OverallResult(m.key, m.name, results)
	}

	// Check service binding.
	fmt.Println("found postgres hostname:" + m.host)
	if m.host == "" {
		fmt.Println("no postgres uri found")
		results = append(results, SmokeTestResult{Name: postgresTestBinding, Result: false, Error: postgresErrorBinding})
		return OverallResult(m.key, m.name, results)
	}
	results = append(results, SmokeTestResult{Name: postgresTestBinding, Result: true})

	// Open connection.
	openConnection := func() (interface{}, error) {
		return sql.Open("pgx", m.uri)
	}
	obj, success := RunTestPart(openConnection, postgresTestConnection, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}
	db := obj.(*sql.DB)
	defer db.Close()

	// Prepare create table.
	prepareCreateTable := func() (interface{}, error) {
		return db.Prepare("CREATE TABLE IF NOT EXISTS deepthought(theanswertoeverything integer)")
	}
	obj, success = RunTestPart(prepareCreateTable, postgresTestPrepareCreate, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}
	createTableStmt := obj.(*sql.Stmt)
	defer createTableStmt.Close()

	// Create table.
	createTable := func() (interface{}, error) {
		return createTableStmt.Exec()
	}
	_, success = RunTestPart(createTable, postgresTestCreate, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}

	// Prepare insert.
	prepareInsert := func() (interface{}, error) {
		return db.Prepare("INSERT INTO deepthought(theanswertoeverything) VALUES($1)")
	}
	obj, success = RunTestPart(prepareInsert, postgresTestPrepareInsert, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}
	insertStmt := obj.(*sql.Stmt)
	defer insertStmt.Close()

	// Insert.
	insert := func() (interface{}, error) {
		return insertStmt.Exec(42)
	}
	_, success = RunTestPart(insert, postgresTestInsert, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}

	// Select.
	query := func() (interface{}, error) {
		return db.Query("SELECT * FROM deepthought WHERE theanswertoeverything = 42")
	}
	_, success = RunTestPart(query, postgresTestSelect, &results)
	if !success {
		return OverallResult(m.key, m.name, results)
	}

	// delete
	deleteQuery := func() (interface{}, error) {
		return db.Query("DELETE FROM deepthought WHERE theanswertoeverything = 42")
	}
	RunTestPart(deleteQuery, postgresTestDelete, &results)

	// Determine overall result and return.
	return OverallResult(m.key, m.name, results)
}

