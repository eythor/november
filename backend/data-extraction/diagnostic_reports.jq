# Extract diagnostic report data from FHIR DiagnosticReport resource and format for SQLite insertion
{
  id: .id,
  resource_type: "DiagnosticReport",
  status: .status,
  category: (.category[0].coding[0].code // null),
  code: (.code.coding[0].code // null),
  display: (.code.coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  performer_id: (.performer[0].reference | split("/")[1] // null),
  effective_datetime: (.effectiveDateTime // null),
  issued_datetime: (.issued // null),
  conclusion: (.conclusion // null),
  raw_json: (. | tostring)
}