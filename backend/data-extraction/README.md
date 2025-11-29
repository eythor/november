# FHIR Data Extraction

This directory contains jq scripts to extract and flatten FHIR resources from NDJSON format into structures suitable for SQLite import.

## Available Extraction Scripts

| Entity | jq File | Description |
|--------|---------|-------------|
| Patient | `patients.jq` | Extract patient demographics and identifiers |
| Practitioner | `practitioners.jq` | Extract healthcare provider information |
| Organization | `organizations.jq` | Extract healthcare organization data |
| Location | `locations.jq` | Extract healthcare facility locations |
| Encounter | `encounters.jq` | Extract healthcare encounters/visits |
| Condition | `conditions.jq` | Extract medical conditions and diagnoses |
| Observation | `observations.jq` | Extract lab results and vital signs |
| Procedure | `procedures.jq` | Extract medical procedures |
| Immunization | `immunizations.jq` | Extract vaccination records |
| AllergyIntolerance | `allergy_intolerances.jq` | Extract allergy and intolerance data |
| MedicationRequest | `medication_requests.jq` | Extract prescription orders |
| DiagnosticReport | `diagnostic_reports.jq` | Extract diagnostic test results |
| ImagingStudy | `imaging_studies.jq` | Extract medical imaging studies |
| Claim | `claims.jq` | Extract insurance claims |
| CareTeam | `care_teams.jq` | Extract care team information |
| DocumentReference | `document_references.jq` | Extract medical document references |

## Usage

### Extract Single Entity
```bash
# Extract to JSON
jq -f encounters.jq ../entities/Encounter.ndjson

# Extract to CSV
jq -r -f encounters.jq ../entities/Encounter.ndjson | jq -r '@csv'
```

### Extract All Entities
```bash
# Run the automated extraction script
./extract-all.sh
```

This will create CSV files in `../extracted-csv/` directory.

### Import to SQLite
```bash
sqlite3 ../database.db
.mode csv
.import extracted-csv/patients.csv patients
.import extracted-csv/encounters.csv encounters
.import extracted-csv/conditions.csv conditions
# ... repeat for other tables
```

## Data Mapping

Each jq script maps FHIR resource fields to the corresponding SQLite table schema defined in `../schema.sql`. Key mappings include:

- **Resource references** are converted to foreign key IDs
- **CodeableConcept** fields are flattened to code and display values
- **Complex structures** are simplified to essential fields
- **Raw JSON** is preserved in the `raw_json` field for full data access

## Notes

- All scripts handle missing/optional fields gracefully with null values
- Reference fields (e.g., `Patient/123`) are split to extract just the ID (`123`)
- Date/time fields are preserved in ISO 8601 format
- Complex nested structures are flattened based on common use patterns
