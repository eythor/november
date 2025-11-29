# Extract care team data from FHIR CareTeam resource and format for SQLite insertion
{
  id: .id,
  resource_type: "CareTeam",
  status: (.status // null),
  category: (.category[0].coding[0].code // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  period_start: (.period.start // null),
  period_end: (.period.end // null),
  raw_json: (. | tostring)
}