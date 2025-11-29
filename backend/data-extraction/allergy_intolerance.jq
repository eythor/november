# Extract allergy intolerance data from FHIR AllergyIntolerance resource and format for SQLite insertion
{
  id: .id,
  resource_type: "AllergyIntolerance",
  clinical_status: (.clinicalStatus.coding[0].code // null),
  verification_status: (.verificationStatus.coding[0].code // null),
  type: (.type // null),
  category: (.category[0] // null),
  criticality: (.criticality // null),
  code: (.code.coding[0].code // null),
  display: (.code.coding[0].display // null),
  patient_id: (.patient.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  recorded_date: (.recordedDate // null),
  recorder_id: (.recorder.reference | split("/")[1] // null),
  raw_json: (. | tostring)
}