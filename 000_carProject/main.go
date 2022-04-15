package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-sql-driver/mysql"
)

var tpl *template.Template

var carinv []carInvSchema

// var pipelineCarInv []carInvSchema

type carInvSchema struct {
	Id          string
	Make        string
	Model       string
	Description string
	Mileage     int
	Price       string
	Term        int
	Provider    string
}

type carResults struct {
	Id          string
	Make        string
	Model       string
	Description string
	Mileage     string
	Price       string
	Term        string
	Provider    string
}

func init() {
	tpl = template.Must(template.ParseGlob("templates/*"))
}

func main() {

	http.HandleFunc("/", index)
	http.HandleFunc("/deals", deals)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		db, err := sql.Open("mysql", "root:Pass2@tcp(127.0.0.1:3306)/carproject")
		if err != nil {
			log.Fatal("Unable to open connection to db")
		}
		defer db.Close()

		//processing form submisson
		//opening file
		file, _, err := req.FormFile("filename")
		if err != nil || file == nil {
			http.Error(w, "Unable to upload file", http.StatusInternalServerError)
		}

		defer file.Close()

		//reading the file
		rdr := csv.NewReader(file)
		rows, err := rdr.ReadAll()
		if err != nil {
			log.Fatalln(err)
		}
		//creating variables to store row values inside of
		var id string
		var make string
		var model string
		var description string
		var mileage string
		var price string
		var term string
		var provider string

		// ranging over the rows
		for i, row := range rows {
			//checking to see if data is loaded inside of an CSV
			if len(row) == 0 {
				http.Error(w, "No information inside of CSV", http.StatusBadRequest)
				break
			}

			// Don't need to store values on first iteration ++i
			if i == 0 {
				continue
			}

			//inserting data based on length of row aka number of columns
			if len(row) == 7 {

				id = row[0]
				make = row[1]
				model = row[2]
				description = row[3]
				mileage = row[4]
				price = "£" + row[5]
				term = row[6]
				provider = "prettygoodcardeals.com"

			} else {

				id = row[0]
				make = strings.ToUpper(row[1])
				model = row[2]
				description = ""
				mileage = strings.Replace(row[5], "k", "000", -1)
				price = row[3]
				term = row[4]
				provider = "amazingcars.co.uk"
			}
			resultsM, _ := strconv.ParseInt(mileage, 10, 0)
			resultsT, _ := strconv.ParseInt(term, 10, 0)

			//storing inside of the struct
			carinv = append(carinv, carInvSchema{
				Id:          id,
				Make:        make,
				Model:       model,
				Description: description,
				Mileage:     int(resultsM),
				Price:       price,
				Term:        int(resultsT),
				Provider:    provider,
			})

		}
		//Preparing what i want to insert into the db
		stmt, err := db.Prepare("INSERT INTO carinv(id, make, model, description, mileage, price, term, provider) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		//ranging over the slice of struct
		for _, car := range carinv {
			//storing each value at the specified key value pair
			_, err := stmt.Exec(car.Id, car.Make, car.Model, car.Description, car.Mileage, car.Price, car.Term, car.Provider)
			var mysqlErr *mysql.MySQLError

			//if the ID is duplicate continue
			//
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				log.Print("Duplicate ID found: ", car.Id)
				//replacing duplicate ID with new associated data
				stmnt2, _ := db.Prepare("REPLACE INTO carinv(id, make, model, description, mileage, price, term, provider) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
				stmnt2.Exec(car.Id, car.Make, car.Model, car.Description, car.Mileage, car.Price, car.Term, car.Provider)
				stmnt2.Close()
				// continue
			} else if err != nil {
				log.Fatal(err)
			}
		}

		//clearing out the carinv variable to keep healthy duplicate logging
		carinv = []carInvSchema{}

		//redirecting to /deals
		http.Redirect(w, req, "/deals", http.StatusSeeOther)
	}

	err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
	check(err)
}

//displaying data
func deals(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	dbData := []carResults{}
	searchResults := make(map[string]string)
	if req.Method == http.MethodGet {
		db, err := sql.Open("mysql", "root:Pass@tcp(127.0.0.1:3306)/carproject")
		if err != nil {
			log.Fatal("Unable to open connection to db")
		}
		defer db.Close()

		query := req.URL.Query()
		makeQuery := query.Get("make")
		if len(makeQuery) > 0 {
			searchResults["make"] = makeQuery
		}
		query = req.URL.Query()
		model := query.Get("model")
		if len(model) > 0 {
			searchResults["model"] = model
		}
		query = req.URL.Query()
		mileage := query.Get("mileage")
		if len(mileage) > 0 {
			searchResults["mileage"] = mileage
		}
		query = req.URL.Query()
		term := query.Get("term")
		if len(term) > 0 {
			searchResults["term"] = term
		}

		args := []string{}
		var addToQuery string
		var queryArray []string
		var stringQuery = "SELECT * FROM carinv"
		for key, value := range searchResults {
			if value != "" {
				addToQuery = key + "=" + " ?"
			}
			queryArray = append(queryArray, addToQuery)
			args = append(args, value)
		}

		if len(args) != 0 {
			stringQuery += " WHERE " + strings.Join(queryArray, " AND ")
		}
		argInterface := make([]interface{}, len(args))
		for i, v := range args {
			argInterface[i] = v
		}
		rows, err := db.Query(stringQuery, argInterface...)
		if err != nil {
			log.Fatalf("Unable to query rows %s", err)
		}

		defer rows.Close()
		for rows.Next() {
			carresults := carResults{}
			err = rows.Scan(&carresults.Id, &carresults.Make, &carresults.Model, &carresults.Description, &carresults.Mileage, &carresults.Price, &carresults.Term, &carresults.Provider)
			check(err)
			dbData = append(dbData, carresults)
		}

	}

	err := tpl.ExecuteTemplate(w, "parseddata.gohtml", dbData)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
