resource "observe_reference_table" "example" {
  label       = "Example"
  source_file = "path/to/reference_table.csv"
  checksum    = filemd5("path/to/reference_table.csv") // must always be filemd5(source_file)
  description = "State Populations"
  primary_key = ["state_code"]
  label_field = "state_name"
}
