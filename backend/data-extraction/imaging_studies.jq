# Extract imaging study data from FHIR ImagingStudy resource and format for SQLite insertion
{
  id: .id,
  resource_type: "ImagingStudy",
  status: .status,
  modality: (.modality[0].code // null),
  patient_id: (.subject.reference | split("/")[1]),
  encounter_id: (.encounter.reference | split("/")[1] // null),
  started_datetime: (.started // null),
  number_of_series: (.numberOfSeries // null),
  number_of_instances: (.numberOfInstances // null),
  procedure_code: (.procedureCode[0].coding[0].code // null),
  procedure_display: (.procedureCode[0].coding[0].display // null),
  raw_json: (. | tostring)
}