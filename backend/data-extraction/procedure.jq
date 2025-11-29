# Extract procedure data from FHIR Procedure resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Procedure",
  status: .status,
  code: (.code.coding[0].code // null),
  display: (.code.coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  performer_id: (.performer[0].actor.reference | split("/")[1] // null),
  performed_datetime: (.performedDateTime // null),
  performed_period_start: (.performedPeriod.start // null),
  performed_period_end: (.performedPeriod.end // null),
  raw_json: (. | tostring)
}