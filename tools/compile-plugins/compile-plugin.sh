#!/bin/sh

# Check if enough arguments are provided
if [ "$#" -lt 2 ]; then
    echo "Error: Not enough arguments provided."
    echo "Usage: $0 <output_filename> <plugins_directory>"
    exit 1
fi

output_file="$1"
plugins_dir="$2"

# Save current directory to return to it later
current_dir=$(pwd)

# Change to plugins directory
cd "$plugins_dir" || { echo "Error: Cannot change to directory $plugins_dir"; exit 1; }

# Create output file with package declaration
printf "package public\n\n" > "$output_file"
printf "import (\n" >> "$output_file"

# Find all directories and generate import statements
find . -mindepth 1 -maxdepth 1 -type d | sort | while read -r dir; do
    c=$(basename "$dir")
    printf "    _ \"github.com/formancehq/payments/internal/connectors/plugins/public/%s\"\n" "$c" >> "$output_file"
done

# Finish import block
printf ")\n" >> "$output_file"

# Return to original directory
cd "$current_dir" || { echo "Error: Cannot return to original directory"; exit 1; }

echo "Successfully generated $output_file with plugin imports"