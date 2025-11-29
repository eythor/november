# Extract location data from FHIR Location resource and format for SQLite insertion
{
  id: .id,
  resource_type: "Location",
  name: (.name // null),
  type: (.type[0].coding[0].display // null),
  address_line: (.address.line[0] // null),
  city: (.address.city // null),
  state: (.address.state // null),
  postal_code: (.address.postalCode // null),
  country: (.address.country // null),
  latitude: (.position.latitude // null),
  longitude: (.position.longitude // null),
  raw_json: (. | tostring)
}