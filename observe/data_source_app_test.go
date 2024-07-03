package observe

// func TestAccObserveDataApp(t *testing.T) {
// 	randomPrefix := acctest.RandomWithPrefix("tf")

// 	resource.ParallelTest(t, resource.TestCase{
// 		PreCheck:  func() { testAccPreCheck(t) },
// 		Providers: testAccProviders,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: fmt.Sprintf(`
// 				resource "observe_workspace" "example" {
// 					name = "%[1]s"
// 				}

// 				resource "observe_folder" "example" {
// 				  workspace = observe_workspace.example.oid
// 				  name      = "%[1]s"
// 				}

// 				resource "observe_datastream" "example" {
// 				  workspace = observe_workspace.example.oid
// 				  name      = "%[1]s"
// 				}

// 				resource "observe_app" "example" {
// 				  folder    = observe_folder.example.oid

// 				  module_id = "observeinc/openweather/observe"
// 				  version   = "0.2.1"

// 				  variables = {
// 					datastream = observe_datastream.example.id
// 					api_key    = "00000000000000000000000000000000"
// 				  }
// 				}

// 				data "observe_app" "example" {
// 				  folder = observe_folder.example.oid
// 				  name   = observe_app.example.name
// 				}`, randomPrefix),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("data.observe_app.example", "module_id", "observeinc/openweather/observe"),
// 					resource.TestCheckResourceAttr("data.observe_app.example", "version", "0.2.1"),
// 				),
// 			},
// 			{
// 				Config: fmt.Sprintf(`
// 				resource "observe_workspace" "example" {
// 					name = "%[1]s"
// 				}

// 				resource "observe_folder" "example" {
// 				  workspace = observe_workspace.example.oid
// 				  name      = "%[1]s"
// 				}

// 				resource "observe_datastream" "example" {
// 				  workspace = observe_workspace.example.oid
// 				  name      = "%[1]s"
// 				}

// 				resource "observe_app" "example" {
// 				  folder    = observe_folder.example.oid

// 				  module_id = "observeinc/openweather/observe"
// 				  version   = "0.2.1"

// 				  variables = {
// 					datastream = observe_datastream.example.id
// 					api_key    = "00000000000000000000000000000000"
// 				  }
// 				}

// 				data "observe_app" "example" {
// 				  id = observe_app.example.id
// 				}`, randomPrefix),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr("data.observe_app.example", "module_id", "observeinc/openweather/observe"),
// 					resource.TestCheckResourceAttr("data.observe_app.example", "version", "0.2.1"),
// 				),
// 			},
// 		},
// 	})
// }
