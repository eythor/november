#!/usr/bin/env bash

# Data Extraction Script
# Converts FHIR NDJSON entities to CSV format for SQLite import

set -e

ENTITIES_DIR="../entities"
OUTPUT_DIR="../extracted-csv"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}FHIR Data Extraction Tool${NC}"
echo -e "${BLUE}=========================${NC}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to extract entity data
extract_entity() {
    local entity="$1"
    local entity_file="$ENTITIES_DIR/${entity}.ndjson"
    local jq_file="${entity,,}.jq"  # Convert to lowercase
    local output_file="$OUTPUT_DIR/${entity,,}.csv"
    
    if [ ! -f "$entity_file" ]; then
        echo -e "${YELLOW}Warning: $entity_file not found, skipping...${NC}"
        return
    fi
    
    if [ ! -f "$jq_file" ]; then
        echo -e "${YELLOW}Warning: $jq_file not found, skipping...${NC}"
        return
    fi
    
    echo -e "${GREEN}Processing $entity...${NC}"
    
    # Count records
    local record_count=$(wc -l < "$entity_file")
    echo "  Records: $record_count"
    
    # Extract data
    jq -r -f "$jq_file" "$entity_file" | jq -r '@csv' > "$output_file"
    
    local extracted_count=$(wc -l < "$output_file")
    echo "  Extracted: $extracted_count records to $output_file"
}

# List of entities to process
ENTITIES=(
    "Patient"
    "Practitioner"
    "Organization"
    "Location"
    "Encounter"
    "Condition"
    "Observation"
    "Procedure"
    "Immunization"
    "AllergyIntolerance"
    "MedicationRequest"
    "DiagnosticReport"
    "ImagingStudy"
    "Claim"
    "CareTeam"
    "DocumentReference"
)

# Process each entity
for entity in "${ENTITIES[@]}"; do
    extract_entity "$entity"
done

echo ""
echo -e "${GREEN}Extraction complete!${NC}"
echo "CSV files are in: $OUTPUT_DIR"
echo ""
echo "To import into SQLite:"
echo "  sqlite3 database.db"
echo "  .mode csv"
echo "  .import extracted-csv/patients.csv patients"
echo "  .import extracted-csv/encounters.csv encounters"
echo "  # ... etc"