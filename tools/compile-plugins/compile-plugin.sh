#!/bin/sh

# Check if enough arguments are provided
if [ "$#" -lt 4 ]; then
    echo "Error: Not enough arguments provided."
    echo "Usage: $0 <output_filename> <plugins_directory> <package_name> <import_prefix>"
    exit 1
fi

output_file="$1"
plugins_dir="$2"
package_name="$3"
import_prefix="$4"

# Save current directory to return to it later
current_dir=$(pwd)

# Change to plugins directory
cd "$plugins_dir" || { echo "Error: Cannot change to directory $plugins_dir"; exit 1; }

# Create output file with package declaration
printf "package %s\n\n" "$package_name" > "$output_file"
printf "import (\n" >> "$output_file"

# Find all directories and generate import statements
find . -mindepth 1 -maxdepth 1 -type d | sort | while read -r dir; do
    c=$(basename "$dir")
    printf "    _ \"%s/%s\"\n" "$import_prefix" "$c" >> "$output_file"
done

# Finish import block
printf ")\n" >> "$output_file"

# Return to original directory
cd "$current_dir" || { echo "Error: Cannot return to original directory"; exit 1; }

echo "Successfully generated $output_file with plugin imports"
