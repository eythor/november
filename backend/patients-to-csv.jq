# Convert FHIR Patient NDJSON to CSV format for SQLite import
# Usage: jq -r -f patients-to-csv.jq Patient.ndjson > patients.csv

def escape_csv:
  if . == null then ""
  else
    tostring | 
    if test("[,\"\n\r]") then
      "\"" + (gsub("\""; "\"\"")) + "\""
    else
      .
    end
  end;

[
  .id,
  "Patient",
  (.name[0].given[0] // null),
  (.name[0].family // null),
  (.name[0].prefix[0] // null),
  .gender,
  .birthDate,
  (.maritalStatus.coding[0].display // null),
  (.telecom[] | select(.system == "phone" and .use == "home") | .value),
  (.address[0].line[0] // null),
  (.address[0].city // null),
  (.address[0].state // null),
  (.address[0].postalCode // null),
  (.address[0].country // null),
  (.identifier[] | select(.type.coding[0].code == "SS") | .value),
  (.identifier[] | select(.type.coding[0].code == "DL") | .value),
  (.identifier[] | select(.type.coding[0].code == "PPN") | .value),
  (.extension[] | select(.url == "http://hl7.org/fhir/us/core/StructureDefinition/us-core-race") | .extension[] | select(.url == "ombCategory") | .valueCoding.display),
  (.extension[] | select(.url == "http://hl7.org/fhir/us/core/StructureDefinition/us-core-ethnicity") | .extension[] | select(.url == "ombCategory") | .valueCoding.display),
  (.extension[] | select(.url == "http://hl7.org/fhir/StructureDefinition/patient-birthPlace") | "\(.valueAddress.city), \(.valueAddress.state)"),
  (.extension[] | select(.url == "http://hl7.org/fhir/StructureDefinition/patient-mothersMaidenName") | .valueString),
  (.communication[0].language.coding[0].code // null),
  (. | tostring)
] | map(escape_csv) | @csv