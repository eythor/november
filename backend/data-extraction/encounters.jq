# Extract encounter data from FHIR Encounter resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Encounter",
  status: .status,
  class: .class.code,
  type_code: (.type[0].coding[0].code // null),
  type_display: (.type[0].coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  practitioner_id: (.participant[] | select(.type[0].coding[0].code == "PPRF") | .individual.reference | split("/")[1]),
  organization_id: (.serviceProvider.reference | split("/")[1] // null),
  location_id: (.location[0].location.reference | split("/")[1] // null),
  start_datetime: .period.start,
  end_datetime: (.period.end // null),
  raw_json: (. | tostring)
}