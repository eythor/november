# Extract observation data from FHIR Observation resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Observation",
  status: .status,
  category: (.category[0].coding[0].code // null),
  code: (.code.coding[0].code // null),
  display: (.code.coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: ((.encounter.reference // null) | if . then split("/")[1] else null end),
  practitioner_id: ((.performer[0].reference // null) | if . then split("/")[1] else null end),
  effective_datetime: (.effectiveDateTime // null),
  issued_datetime: (.issued // null),
  value_quantity: (.valueQuantity.value // null),
  value_unit: (.valueQuantity.unit // null),
  value_string: (.valueString // .valueCodeableConcept.coding[0].display // null),
  raw_json: (. | tostring)
}