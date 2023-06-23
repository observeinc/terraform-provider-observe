terraform {
  backend "s3" {
    bucket = "observeinc-terraform-state"
    region = "us-west-2"
    key    = "observe-eng.com/127814973959"
  }
}
