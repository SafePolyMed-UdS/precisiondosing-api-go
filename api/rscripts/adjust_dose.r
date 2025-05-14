# -----------------------------------
# Description: Main script for adjusting doses
# Author     : Dominik Selzer
# Date       : 2025-04-17
# Notes      : - Calls the helper functions to read and write orders
#              - Loads libraries and sets up the environment
# -----------------------------------
source("helpers/return-helpers.R")

main <- function() {
  source("helpers/io-helpers.R")
  dbg_out("Starting adjust_dose script...")
  source("startup/packages.R")
  source("report/create_report.R")
  source("helpers/return-test.R")
  source("helpers/service-adjust_dose.R")
  source("settings.R")

  load_packages()

  # Parse Input
  # -----------------------------------
  settings <- read_settings()

  dbg_out("Reading API settings...")
  API_SETTINGS <- create_settings(settings$model_path)

  dbg_out("Reading order...")
  order <- read_order(settings)
  API_SETTINGS$VALUES$model_defaults$SIM_CORES <- settings$r_worker

  # -----------------------------------
  # HANDLING
  # -----------------------------------

  # Case 1: Pre-Check Error
  # -----------------------------------
  ## CONDITION: settings$adjust_dose == FALSE
  ## CREATE ERROR PDF AND RETURN
  if (!settings$adjust_dose) {
    dbg_out("Precheck error - creating error PDF...")
    res <- precheck_error_pdf(order, settings, API_SETTINGS)
  }

  # Case 2: Simulate
  # -----------------------------------

  ## CONDITION: settings$adjust_dose == TRUE
  ## SIMULATE, POSTPROCESS AND CREATE REPORT
  ## ON ERROR: CREATE ERROR PDF AND RETURN
  if (settings$adjust_dose) {
    dbg_out("Running dose adjustment...")
    res <- execute(order, settings, API_SETTINGS)
  }
  dbg_out(toJSON(res, auto_unbox = TRUE))
  return(res)
}

safeReturn(main)
