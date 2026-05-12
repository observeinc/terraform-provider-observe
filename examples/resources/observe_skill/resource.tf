# Inline skill content
resource "observe_skill" "investigate_high_cpu" {
  label       = "Investigate High CPU"
  description = "Step-by-step runbook for investigating high CPU usage on application servers."

  content = <<-EOT
    # Investigate High CPU

    When triggered by a high CPU alert, follow these steps:

    1. Identify the affected host and process using top/htop data
    2. Check if the spike correlates with a recent deployment
    3. Look for abnormal request patterns or traffic spikes
    4. Review application logs for errors or unusual activity
    5. Check memory and disk I/O for cascading resource pressure
  EOT
}

# Load skill content from a file
resource "observe_skill" "db_troubleshooting" {
  label       = "Database Troubleshooting"
  description = "Runbook for investigating database performance issues."
  content     = file("${path.module}/skills/db-troubleshooting.md")
  # Optional: visibility = "Private" for the authenticated user only; default is "Workspace"
}
