# Extract condition data from FHIR Condition resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Condition",
  clinical_status: (.clinicalStatus.coding[0].code // null),
  verification_status: (.verificationStatus.coding[0].code // null),
  category: (.category[0].coding[0].code // null),
  code: (.code.coding[0].code // null),
  display: (.code.coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  onset_datetime: (.onsetDateTime // null),
  recorded_date: (.recordedDate // null),
  abatement_datetime: (.abatementDateTime // null),
  raw_json: (. | tostring)
}