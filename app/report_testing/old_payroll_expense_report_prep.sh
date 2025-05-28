#!/bin/bash

# Define source and destination directories
SOURCE_DIR="expenses_payroll/old_unmodified"
DEST_DIR="expenses_payroll/old_preprocessed"

# Check if source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
  echo "Source directory '$SOURCE_DIR' not found."
  exit 1
fi

# Check if destination directory exists, if so, clear its contents
if [ -d "$DEST_DIR" ]; then
  echo "Clearing contents of '$DEST_DIR'..."
  rm -rf "${DEST_DIR:?}"/*
else
  # Create destination directory if it doesn't exist
  echo "Creating directory '$DEST_DIR'..."
  mkdir -p "$DEST_DIR"
fi

# Copy all CSV files from source to destination
echo "Copying CSV files from '$SOURCE_DIR' to '$DEST_DIR'..."
cp "$SOURCE_DIR"/*.csv "$DEST_DIR/"

# Process each CSV file in the destination directory
echo "Processing files in '$DEST_DIR'..."
for file in "$DEST_DIR"/*.csv; do
  if [ -f "$file" ]; then
    echo "Processing $file..."
    # Trim trailing whitespace from Description field
    mlr --csv put 'for (k in $*) { 
      $Description = gsub($Description, " +$", "")
    }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
  fi
done

echo "Script finished."
