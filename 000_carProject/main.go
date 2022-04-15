package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-sql-driver/mysql"
)

var tpl *template.Template

var carinv []carInvSchema

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
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("failed to start the server: %v", err)
	}
}

func index(w http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		db, err := sql.Open("mysql", "root:pass@tcp(127.0.0.1:3306)/carproject")
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

		//clearing out the carinv variable to keep healthy logging
		carinv = []carInvSchema{}

		//redirecting to /deals
		http.Redirect(w, req, "/deals", http.StatusSeeOther)
	}

	err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
	if err != nil {
		log.Fatalln()
	}
}

//displaying data
func deals(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	//struct for data to be passed into
	dbData := []carResults{}
	searchResults := make(map[string]string)
	if req.Method == http.MethodGet {
		db, err := sql.Open("mysql", "root:pass@tcp(127.0.0.1:3306)/carproject")
		if err != nil {
			log.Fatalf("Unable to open connection to db %s", err)
		}
		defer db.Close()

		// GET QUERY
		query := req.URL.Query()
		//STORE QUERY
		makeQuery := query.Get("make")
		if len(makeQuery) > 0 {
			//STORIND INSIDE OF MAP
			searchResults["make"] = makeQuery
		}
		// GET QUERY
		query = req.URL.Query()
		//STORE QUERY
		model := query.Get("model")
		if len(model) > 0 {
			//STORIND INSIDE OF MAP
			searchResults["model"] = model
		}
		// GET QUERY
		query = req.URL.Query()
		//STORE QUERY
		mileage := query.Get("mileage")
		if len(mileage) > 0 {
			//STORING INSIDE OF MAP
			searchResults["mileage"] = mileage
		}
		// GET QUERY
		query = req.URL.Query()
		//STORE QUERY
		term := query.Get("term")
		if len(term) > 0 {
			//STORING INSIDE OF MAP
			searchResults["term"] = term
		}

		//CREATING A SLICE TO STORE THE VALUES INSIDE OF
		args := []string{}

		//CREATING A PORTION OF THE QUERY STRING TO CONCATENATE AND STORE INSIDE OF THE ARRAY
		var addToQuery string

		//CREATING AN ARRAY TO STORE addToQuery
		var queryArray []string

		//BASE STRING QUERY TO BUILD OFF OF
		var stringQuery = "SELECT * FROM carinv"

		//RANGING OVER MAP AND CREATING MORE VARIABLE FOR THE DB
		for key, value := range searchResults {
			if value != "" {
				addToQuery = key + "=" + " ?"
			}
			//appending each sql param inside of an array
			queryArray = append(queryArray, addToQuery)
			//passing values into the slice
			args = append(args, value)
		}

		//CREATING THE STRING QUERY FOR THE DB DESIGN
		if len(args) != 0 {
			stringQuery += " WHERE " + strings.Join(queryArray, " AND ")
		}

		//CONVERTING ARGS TO VARIADIC PARAMETER OF ANY TIME
		argInterface := make([]interface{}, len(args))
		for i, v := range args {
			//WRITING THE VALUES INTO ARGINTERFACE
			argInterface[i] = v
		}

		//PASSING THE QUERY INTO ROWS
		rows, err := db.Query(stringQuery, argInterface...)
		if err != nil {
			log.Fatalf("Unable to query rows %s", err)
		}
		//CLOSE AT END OF CODE BLOCK
		defer rows.Close()

		//LOOP OVER ROWS IF TRUE CONTINUE
		for rows.Next() {
			//INITIALIZING VARIABLE TO STORE DB VALUES INTO
			carresults := carResults{}
			//SCANNING VALUES INTO carrresults
			err = rows.Scan(&carresults.Id, &carresults.Make, &carresults.Model, &carresults.Description, &carresults.Mileage, &carresults.Price, &carresults.Term, &carresults.Provider)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			//APPENDING THE DATA TO BE PIPELINED INTO THE TEMPLATE
			dbData = append(dbData, carresults)
		}

	}

	//EXECUTING THE TEMPLATE AND PASSING THE DBDATA
	err := tpl.ExecuteTemplate(w, "parseddata.gohtml", dbData)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}
