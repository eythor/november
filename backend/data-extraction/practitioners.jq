# Extract practitioner data from FHIR Practitioner resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Practitioner",
  given_name: (.name[0].given[0] // null),
  family_name: (.name[0].family // null),
  prefix: (.name[0].prefix[0] // null),
  gender: (.gender // null),
  address_line: (.address[0].line[0] // null),
  city: (.address[0].city // null),
  state: (.address[0].state // null),
  postal_code: (.address[0].postalCode // null),
  country: (.address[0].country // null),
  raw_json: (. | tostring)
}