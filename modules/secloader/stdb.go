package secloader

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// database connector for getting statements:
// main lookup table: convert three cid (cid0,cid1,cid2) -> company table
// 		columns:
// 			cid0,cid1,cid2: key of this table
//			adsh: string of company's adsh
//		company table:
// 			name: adsh XXXXXXXXXX-XX-XXXXXX (10-2-6) unique company id
// 			columns:
// 				date: year and quarterly of a row
// 				others: ...
// second record table: record companies updated in a single period
//		columns:
// 			date: year and quarterly of a company
// 			name: adsh XXXXXXXXXX-XX-XXXXXX (10-2-6) unique company id

var db *sql.DB

const serverAddr = "127.0.0.1"
const serverPort = "3306"
const dbUser = "root"
const dbPW = "inno2021"
const dbName = "test"

func init() {
	// <UserName>:<UserPW>@tcp(<Address>:<Port>)/<DBName>
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPW, serverAddr, serverPort, dbUser)
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatal("failed to connect to %s: %s.\n", dataSourceName, err)
	}
	// test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("failed to connect to %s: %s.\n", dataSourceName, err)
	}
	// initialize all_companies table
	rows, err := db.Query("SELECT * FROM all_companies")
	if err != nil {
		dbcmd := `CREATE TABLE all_companies 
		(
			cid0 INT UNSIGNED NOT NULL,
			cid1 INT UNSIGNED NOT NULL,
			cid2 INT UNSIGNED NOT NULL,
			adsh CHAR(255) DEFAULT '',
			primary key (cid0,cid1,cid2)
		);`
		r, err := db.Exec(dbcmd)
		if err != nil {
			log.Fatal("failed to create table all_companies: %s.\n", err)
		}
	}
	rows.Close()
	// initialize statement_time table
	rows, err = db.Query("SELECT * FROM statement_time")
	if err != nil {
		dbcmd := `CREATE TABLE statement_time 
		(
			date DATE NOT NULL,
			adsh CHAR(255) DEFAULT ''
		);`
		r, err := db.Exec(dbcmd)
		if err != nil {
			log.Fatal("failed to create table statement_time: %s.\n", err)
		}
	}
	rows.Close()
}

func ReadByCompanyID(cid CompanyID) ([]AccountTab, error) {
	rows, err := db.Query("SELECT * FROM $1", cid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		// handle err
		if isdebug {
			log.Println("Error: failed to get columns of table %s: %s.", cid, err)
		}
		return nil, err
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		if isdebug {
			log.Println("Error: failed to get columns of table %s: %s.", cid, err)
		}
		return nil, err
	}
	ptrs := make([]interface{}, len(colTypes))
	datas := make([]interface{}, len(colTypes))
	for i, v := range colTypes {
		// "VARCHAR", "TEXT", "NVARCHAR", "DECIMAL", "BOOL", "INT", and "BIGINT".
		typN := strings.ToUpper(v.DatabaseTypeName())
		// switch didnot provide top N comparison hence we use if
		if typN[:3] == "INT" {
			datas[i] = int(0)
			ptrs[i] = &datas[i]
		} else if typN[:5] == "FLOAT" {
			datas[i] = float32(0.0)
			ptrs[i] = &datas[i]
		} else {
			datas[i] = ""
			ptrs[i] = &datas[i]
		}
		// ptrs[i] = reflect.TypeOf()
	}

	var forms = []AccountTab{}
	for rows.Next() {
		form := make(AccountTab)
		err = rows.Scan(ptrs...)
		if err != nil && isdebug {
			t := ""
			rows.Scan(&t)
			log.Println("Warring: fail to scan %s: %s.", t, err)
		}
		for i, v := range colNames {
			form[v] = datas[i]
		}
		forms = append(forms, form)
	}
	return forms, nil
}

func CloseDB() { db.Close() }

func getTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		// INT
		return "INT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		// UINT
		return "INT UNSIGNED"
	case reflect.Bool:
		// BOOL
		return "BOOL"
	case reflect.String:
		return "TEXT"
	default:
		return "TEXT"
	}
}

func autoFmt(v interface{}) string {
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		// INT
		return fmt.Sprintf("%d", v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		// UINT
		return fmt.Sprintf("%d", v)
	case reflect.Bool:
		// BOOL
		return fmt.Sprint(v)
	case reflect.String:
		return fmt.Sprint(v)
	default:
		return fmt.Sprint(v)
	}
}

func updateDB(cid CompanyID, year int, quarterly int, datas AccountTab) {
	var adsh = cid.String()
	var dbcmd string = ""
	// 1. update all_companies table
	rows, err := db.Query("SELECT * FROM $1;", adsh)
	if err != nil {
		// company didn't exist
		dbcmd = `INSERT INTO all_companies (cid0, cid1, cid2, adsh)
		VALUES ($1, $2, $3, '$4')`
		_, err := db.Exec(dbcmd, cid.ID0, cid.ID1, cid.ID2, adsh)

		if err != nil {
			if isdebug {
				log.Println("Error: failed to create table %s: %s.", adsh, err)
			}
			return
		}

		dbcmd = `CREATE TABLE $1 
		(
			date DATE NOT NULL,
			currency STRING, 
		);`
		db.Exec(dbcmd, adsh)
		rows, err = db.Query("SELECT * FROM $1;", adsh)
		if err != nil {
			if isdebug {
				log.Println("Error: failed to find table %s: %s.", adsh, err)
			}
			return
		}
	}
	defer rows.Close()

	// 2. update company table
	colNames, err := rows.Columns()
	// since statement table has hundred-level attributes,
	// we convert existed attributes into set via map[string] interface{}
	// then update attributes didn't exist
	var nameSet = make(map[string]interface{})
	for _, key := range colNames {
		nameSet[key] = nil
	}
	// gathering keys
	var newKeys = []string{}
	var newTypes = []string{}
	for key, v := range datas {
		_, ok := nameSet[key]
		if !ok {
			newKeys = append(newKeys, key)
			newTypes = append(newTypes, getTypeName(reflect.TypeOf(v)))
		}
	}
	// add columns to table
	if len(newKeys) > 0 {
		// construct cmd
		dbcmd = "ALTER TABLE " + adsh + "\n"
		for i, key := range newKeys {
			dbcmd += fmt.Sprintf("ADD %s %s,\n", key, newTypes[i])
		}
		// ..., -> ...;
		dbcmd = dbcmd[:len(dbcmd)-1] + ";"
		_, err := db.Exec(dbcmd)
		if err != nil {
			if isdebug {
				log.Println("Error: failed to alter table %s: %s.", adsh, err)
			}
			return
		}
	}
	// insert row
	dbcmd = fmt.Sprintf("INSERT INTO %s (", adsh)
	var dbvalues = "VALUES ("
	for key, val := range datas {
		dbcmd += key + ","
		dbvalues += autoFmt(val) + ","
	}
	dbcmd = dbcmd[:len(dbcmd)-1] + ")" + dbvalues[:len(dbvalues)-1] + ")"
	_, err = db.Exec(dbcmd)
	if err != nil {
		if isdebug {
			log.Println("Error: failed to insert row to %s: %s.", adsh, err)
		}
		return
	}

	// 3. update statement_time table
	// date:DATE and adsh:String
	// insert date using cmd "DATE 'YYYY-MM-DD'"
	dbcmd = `INSERT INTO statement_time (date, adsh)
	VALUES (DATE '$1-$2-01', '$3')`
	_, err = db.Exec(dbcmd, year, QuartToMonth(quarterly), adsh)
	if err != nil {
		if isdebug {
			log.Println("Error: failed to insert row to tatement_time: %s.", err)
		}
		return
	}
	return
}
