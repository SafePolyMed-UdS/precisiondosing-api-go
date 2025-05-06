# -----------------------------------
# Description: Main script for adjusting doses
# Author     : Dominik Selzer
# Date       : 2025-04-17
# Notes      : - Calls the helper functions to read and write orders
#              - Loads libraries and sets up the environment
# -----------------------------------
source("helpers/return-helpers.R")
debugging <- FALSE

main <- function() {
  source("startup/packages.R")
  source("report/create_report.R")
  source("helpers/io-helpers.R")
  source("helpers/return-test.R")
  source("helpers/service-adjust_dose.R")
  source("settings.R")

  load_packages()
  if (debugging) readRenviron(".env")

  # Parse Input
  # -----------------------------------
  settings <- list(
    mysql_host = get_env_or_stop("R_MYSQL_HOST"),
    mysql_user = get_env_or_stop("R_MYSQL_USER"),
    mysql_password = get_env_or_stop("R_MYSQL_PASSWORD"),
    mysql_db = get_env_or_stop("R_MYSQL_DB"),
    mysql_table = get_env_or_stop("R_MYSQL_TABLE"),
    r_worker = get_env_or_stop("R_WORKER"),
    id = 3,
    adjust_dose = TRUE,
    error_msg = ""
  )
  if (!debugging) {
    settings <- read_settings()
  }
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
    precheck_error_pdf(order, settings)
  }

  # Case 2: Simulate
  # -----------------------------------

  ## CONDITION: settings$adjust_dose == TRUE
  ## SIMULATE, POSTPROCESS AND CREATE REPORT
  ## ON ERROR: CREATE ERROR PDF AND RETURN
  if (settings$adjust_dose) {
    execute(order, settings)
  }
}

safeReturn(main)
