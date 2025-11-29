# Extract organization data from FHIR Organization resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Organization",
  name: (.name // null),
  type: (.type[0].coding[0].display // null),
  phone: (.telecom[] | select(.system == "phone") | .value // null),
  address_line: (.address[0].line[0] // null),
  city: (.address[0].city // null),
  state: (.address[0].state // null),
  postal_code: (.address[0].postalCode // null),
  country: (.address[0].country // null),
  raw_json: (. | tostring)
}