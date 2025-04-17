package main

import (
	"flag"
	"fmt"
	"imports/extract"
	"imports/load"
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
		load.FromParquet(
			"./parquet/Clients.parquet",
			"../app/test_pb_data/data.db",
			"clients",
			map[string]string{
				"id":   "id",
				"name": "name",
			},
		)
	}
}
