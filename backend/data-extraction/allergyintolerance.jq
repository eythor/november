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
  patient_id: (if .patient.reference then .patient.reference | split("/")[1] else null end),
  encounter_id: (if .encounter.reference then .encounter.reference | split("/")[1] else null end),
  recorded_date: (.recordedDate // null),
  recorder_id: (if .recorder.reference then .recorder.reference | split("/")[1] else null end),
  raw_json: (. | tostring)
}