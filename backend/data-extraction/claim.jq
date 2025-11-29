# Extract claim data from FHIR Claim resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Claim",
  status: .status,
  type: (.type.coding[0].code // null),
  use: .use,
  patient_id: (.patient.reference | split("/")[1]),
  provider_id: (.provider.reference | split("/")[1] // null),
  priority: (.priority.coding[0].code // null),
  created_datetime: (.created // null),
  billable_period_start: (.billablePeriod.start // null),
  billable_period_end: (.billablePeriod.end // null),
  total_amount: (.total.value // null),
  currency: (.total.currency // null),
  raw_json: (. | tostring)
}