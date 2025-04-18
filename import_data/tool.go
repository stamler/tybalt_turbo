package main

import (
	"flag"
	"fmt"
	"imports/extract"
	"imports/load"

	"github.com/pocketbase/dbx"
)

// This file is used to run either an export or an import.

func main() {
	// Parse command line arguments
	exportFlag := flag.Bool("export", false, "Export data to Parquet files")
	importFlag := flag.Bool("import", false, "Import data from Parquet files")
	flag.Parse()

	if *exportFlag {
		fmt.Println("Exporting data to Parquet files...")
		extract.ToParquet()
	}

	if *importFlag {
		fmt.Println("Importing data from Parquet files...")

		// --- Load Clients ---
		// Define the specific SQL for the clients table
		clientInsertSQL := "INSERT INTO clients (id, name) VALUES ({:id}, {:name})"

		// Define the binder function for the Client type
		clientBinder := func(item load.Client) dbx.Params {
			return dbx.Params{
				"id":   item.Id,
				"name": item.Name,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Clients.parquet",
			"../app/test_pb_data/data.db",
			"clients",       // Table name (for logging)
			clientInsertSQL, // The specific INSERT SQL
			clientBinder,    // The specific binder function
		)

		// --- Load Contacts ---
		// Define the specific SQL for the contacts table
		contactInsertSQL := "INSERT INTO client_contacts (id, surname, given_name, client) VALUES ({:id}, {:surname}, {:given_name}, {:client})"

		// Define the binder function for the Contact type
		contactBinder := func(item load.ClientContact) dbx.Params {
			return dbx.Params{
				"id":         item.Id,
				"surname":    item.Surname,
				"given_name": item.GivenName,
				"client":     item.Client,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Contacts.parquet",
			"../app/test_pb_data/data.db",
			"client_contacts", // Table name (for logging)
			contactInsertSQL,  // The specific INSERT SQL
			contactBinder,     // The specific binder function
		)

		// --- Load Jobs ---
		// Define the specific SQL for the jobs table
		jobInsertSQL := "INSERT INTO jobs (id, number, description, client, contact, manager) VALUES ({:id}, {:number}, {:description}, {:client}, {:contact}, {:manager})"

		// Define the binder function for the Job type
		jobBinder := func(item load.Job) dbx.Params {
			return dbx.Params{
				"id":          item.Id,
				"number":      item.Number,
				"description": item.Description,
				"client":      item.Client,
				"contact":     item.Contact,
				"manager":     item.Manager,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Jobs.parquet",
			"../app/test_pb_data/data.db",
			"jobs",       // Table name (for logging)
			jobInsertSQL, // The specific INSERT SQL
			jobBinder,    // The specific binder function
		)

	}
}
