# -----------------------------------
# Description: Test script for the report generation
# Author     : Simeon Rüdesheim
# Date       : 2025-04-22
# Notes      : -
#              -
# -----------------------------------

# Dependencies
# -----------------------------------
source("startup/packages.R")
source("helpers.R")
source("settings.R")
source("report/create_report.R")
source("service-adjust_dose.R")

load_packages()
run <- FALSE

# Code
# -----------------------------------
order <- list(id = 1, order_id = 1, order = NULL)

module_data <- readRDS("module_data.rds")
clinical_conc <- module_data$user_data$clinical_conc$table
module_data$user_data$clinical_conc$table <- clinical_conc |>
  slice(0)

output_data <- predictionOutputData()

# Simulate and postprocess
# -----------------------------------
individual <- readRDS("individual.rds")
if (run) {
  dose_sim_res <- dose_sim_routine_api(
    module_data = module_data,
    output_data = output_data,
    settings = API_SETTINGS,
    individual = individual,
    use_fake_data = FALSE,
    create_fake_data = FALSE
  )
  output_data$map_pk$data <- dose_values$map_data
  output_data$map_state <- dose_values$map_state
  output_data$map_pk$clinical_data <- dose_values$clinical_data
  output_data$dose_pk$data <- dose_values$dose_data
  output_data$dose_pk$clinical_data <- module_data$user_data$clinical_conc

  saveRDS(output_data, "output_data.rds")
} else {
  output_data <- readRDS("output_data.rds")
}

# Create report
# -----------------------------------
x <- render_success_pdf(
  user_data = module_data$user_data,
  order = order,
  dose_pk = output_data$dose_pk$data,
  api_settings = API_SETTINGS
)
print(x)
