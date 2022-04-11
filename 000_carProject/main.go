package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/go-sql-driver/mysql"
)

var tpl *template.Template

var carinv []carInvSchema
var pipelineCarInv []carInvSchema

type carInvSchema struct {
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
	http.HandleFunc("/uploadCSV", parsedCSV)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		db, err := sql.Open("mysql", "root:KDaisy!22@tcp(127.0.0.1:3306)/carproject")
		if err != nil {
			log.Fatal("Unable to open connection to db")
		}
		defer db.Close()

		//processing form submisson
		//opening file
		file, _, err := req.FormFile("filename")
		if err != nil {
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
				make = row[1]
				model = row[2]
				description = ""
				mileage = strings.Replace(row[5], "k", "000", -1)
				price = row[3]
				term = row[4]
				provider = "amazingcars.co.uk"
			}

			//storing inside of the struct
			carinv = append(carinv, carInvSchema{
				Id:          id,
				Make:        make,
				Model:       model,
				Description: description,
				Mileage:     mileage,
				Price:       price,
				Term:        term,
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
				continue
			} else if err != nil {
				log.Fatal(err)
			}
		}
		//data to pipeline inside of /uploadCSV
		pipelineCarInv = carinv

		//clearing out the carinv variable to keep healthy duplicate logging
		carinv = []carInvSchema{}

		//redirecting to /uploadCSV
		http.Redirect(w, req, "/uploadCSV", http.StatusSeeOther)
	}

	err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}

//displaying data
func parsedCSV(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tpl.ExecuteTemplate(w, "parseddata.gohtml", pipelineCarInv)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}
