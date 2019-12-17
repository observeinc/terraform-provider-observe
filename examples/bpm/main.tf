provider observe {}

variable "workspace_id" {
  type    = string
}

data "observe_dataset" "observation_table" {
  workspace = var.workspace_id
  name      = "Observation"
}

resource "observe_transform" "http_posts" {
  workspace = var.workspace_id

  stage {
    import = data.observe_dataset.observation_table.id
    pipeline = <<-EOF
      filter OBSERVATION_KIND="httpjson"
      colmake path:string(EXTRA.path)
    EOF
  }

  dataset {
    name = "HTTP Requests"
  }
}

resource "observe_transform" "http_endpoint" {
  workspace = var.workspace_id

  stage {
    name   = "http_posts"
    import = observe_transform.http_posts.id
  }

  stage {
    pipeline = <<-EOF
      filter regex_match(string(fields), /batchelor/)
      timechart duration(seconds(60)), path, bpm:count(fields)
      setvt
      coldrop @."_c_valid_to"
    EOF
  }

  stage {
    linked   = "http_posts"
    pipeline = <<-EOF
      timechart duration(seconds(60)), path, rps:count(1)/60
      mergeevent path=@stage1.path, bpm:@stage1.bpm
    EOF
  }

  dataset {
    name      = "Collector Observation Endpoint"
    freshness = "10s"
  }
}

