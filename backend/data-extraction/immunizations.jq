# Extract immunization data from FHIR Immunization resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Immunization",
  status: .status,
  vaccine_code: (.vaccineCode.coding[0].code // null),
  vaccine_display: (.vaccineCode.coding[0].display // null),
  patient_id: (.patient.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  performer_id: (.performer[0].actor.reference | split("/")[1] // null),
  occurrence_datetime: (.occurrenceDateTime // null),
  primary_source: (.primarySource // null),
  lot_number: (.lotNumber // null),
  expiration_date: (.expirationDate // null),
  raw_json: (. | tostring)
}