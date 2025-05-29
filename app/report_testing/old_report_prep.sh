#!/bin/bash

# Usage: ./old_report_prep.sh <base_dir> <processing_type>
# Processing types: expense, payroll_time, weekly_time

# Check if correct number of arguments provided
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <base_dir> <processing_type>"
    echo "Processing types: expense, payroll_time, weekly_time"
    echo "Expected structure: <base_dir>/old_unmodified/ (source)"
    echo "Will create: <base_dir>/old_preprocessed/ (destination)"
    exit 1
fi

# Assign parameters
BASE_DIR="$1"
PROCESSING_TYPE="$2"

# Set source and destination directories
SOURCE_DIR="$BASE_DIR/old_unmodified"
DEST_DIR="$BASE_DIR/old_preprocessed"

# Validate processing type
case "$PROCESSING_TYPE" in
    expense|payroll_time|weekly_time)
        ;;
    *)
        echo "Invalid processing type: $PROCESSING_TYPE"
        echo "Valid types: expense, payroll_time, weekly_time"
        exit 1
        ;;
esac

# Check if base directory exists
if [ ! -d "$BASE_DIR" ]; then
  echo "Base directory '$BASE_DIR' not found."
  exit 1
fi

# Check if source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
  echo "Source directory '$SOURCE_DIR' not found."
  exit 1
fi

# Remove destination directory if it exists (cleanup from previous runs)
if [ -d "$DEST_DIR" ]; then
  echo "Removing existing '$DEST_DIR'..."
  rm -rf "$DEST_DIR"
fi

# Create destination directory
echo "Creating directory '$DEST_DIR'..."
mkdir -p "$DEST_DIR"

# Copy all CSV files from source to destination
echo "Copying CSV files from '$SOURCE_DIR' to '$DEST_DIR'..."
cp "$SOURCE_DIR"/*.csv "$DEST_DIR/"

# Function to process expense files
process_expense() {
    local file="$1"
    mlr --csv put 'for (k in $*) { 
      $Description = gsub($Description, " +$", "");
      if (typeof($[k]) == "string") {
        $[k] = gsub($[k], "Joe Za", "Joseph Za")
      }
    }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
}

# Function to process payroll time files
process_payroll_time() {
    local file="$1"
    mlr --csv put '
      for (k, v in $*) {
        if (v == "" && is_numeric_field(k)) {
          $[k] = 0
        }
        if (typeof($[k]) == "string") {
          $[k] = gsub($[k], "Joe Za", "Joseph Za")
        }
        $salary = gsub($salary, "true", "TRUE");
        $salary = gsub($salary, "false", "FALSE");
      }

      func is_numeric_field(k) {
        return (k == "Bereavement" || k == "Stat Holiday" || k == "PPTO" || k == "Sick" || k == "Vacation")
      }
    ' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
}

# Function to process weekly time files
process_weekly_time() {
    local file="$1"
    mlr --csv put '
      for (k, v in $*) {
        if (v == "") { $[k] = "0000_first" }
        $client = gsub($client, " +$", "");
        if (typeof($[k]) == "string") {
          $[k] = gsub($[k], "Joe Za", "Joseph Za");
          $[k] = gsub($[k], "Chris Yl", "Christopher Yl")
        }
      }
    ' "$file" | mlr --csv sort -n -f year,month,date,timetype,job,division,qty,nc,surname,givenName | mlr --csv put '
      for (k, v in $*) {
        if (v == "0000_first") { $[k] = "" }
      }
    ' > "${file}.tmp" && mv "${file}.tmp" "$file"
}

# Process each CSV file in the destination directory
echo "Processing files in '$DEST_DIR' with $PROCESSING_TYPE processing..."
for file in "$DEST_DIR"/*.csv; do
  if [ -f "$file" ]; then
    echo "Processing $file..."
    case "$PROCESSING_TYPE" in
        expense)
            process_expense "$file"
            ;;
        payroll_time)
            process_payroll_time "$file"
            ;;
        weekly_time)
            process_weekly_time "$file"
            ;;
    esac
  fi
done

echo "Script finished." 