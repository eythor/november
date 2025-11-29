# Extract document reference data from FHIR DocumentReference resource and format for SQLite insertion
{
  id: .id,
  resource_type: "DocumentReference",
  status: .status,
  type_code: (.type.coding[0].code // null),
  type_display: (.type.coding[0].display // null),
  category: (.category[0].coding[0].code // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.context.encounter[0].reference | split("/")[1] // null),
  author_id: (.author[0].reference | split("/")[1] // null),
  created_datetime: (.date // null),
  content_type: (.content[0].attachment.contentType // null),
  raw_json: (. | tostring)
}