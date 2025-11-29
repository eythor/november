# Extract medication request data from FHIR MedicationRequest resource and format for SQLite insertion
{
  id: .id,
  resource_type: "MedicationRequest",
  status: .status,
  intent: .intent,
  medication_id: (.medicationReference.reference | split("/")[1] // null),
  medication_code: (.medicationCodeableConcept.coding[0].code // null),
  medication_display: (.medicationCodeableConcept.coding[0].display // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  requester_id: (.requester.reference | split("/")[1] // null),
  authored_on: (.authoredOn // null),
  dosage_text: (.dosageInstruction[0].text // null),
  dispense_quantity: (.dispenseRequest.quantity.value // null),
  dispense_unit: (.dispenseRequest.quantity.unit // null),
  raw_json: (. | tostring)
}