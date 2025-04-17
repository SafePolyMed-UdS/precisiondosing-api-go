source("startup/packages.R")
source("helpers.R")
source("settings.R")
source("report/create_report.R")

load_packages()

order <- list(id = 1, order_id = 1, order = NULL)
sim_results <- list(
  order = order,
  errors = list(
    "Some error message1",
    "Some error message2"
  )
)

render_error_pdf(
  results = sim_results,
  settings = SETTINGS
)
