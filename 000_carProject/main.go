package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"text/template"
)

var tpl *template.Template
var carInv []carInvSchema

type carInvSchema struct {
	Id          string
	Make        string
	Model       string
	Description string
	Mileage     string
	Price       string
	Term        string
	Source      string
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
		var source string
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

			//inserting data based on length of row
			if len(row) == 7 {

				id = row[0]
				make = row[1]
				model = row[2]
				description = row[3]
				mileage = row[4]
				price = row[5]
				term = row[6]
				source = "prettygoodcardeals.com"

			} else {

				id = row[0]
				make = row[1]
				model = row[2]
				description = ""
				price = row[3]
				term = row[4]
				mileage = row[5]
				source = "amazingcars.co.uk"
			}

			//storing inside of the struct
			carInv = append(carInv, carInvSchema{
				Id:          id,
				Make:        make,
				Model:       model,
				Description: description,
				Mileage:     mileage,
				Price:       price,
				Term:        term,
				Source:      source,
			})

		}
		http.Redirect(w, req, "/uploadCSV", http.StatusSeeOther)
	}

	err := tpl.ExecuteTemplate(w, "index.gohtml", nil)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}

func parsedCSV(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tpl.ExecuteTemplate(w, "parseddata.gohtml", carInv)
	if err != nil {
		log.Fatal("Unable to execute template")
	}
}
