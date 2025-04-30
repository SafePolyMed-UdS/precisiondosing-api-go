# -----------------------------------
# Description: Contains helper functions for the main script
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# Notes      : - get_env_or_stop: Retrieves an environment variable or stops execution if not found
#              - read_settings: Reads settings from environment variables and command line arguments
#              - read_order: Reads an order from a MySQL database based on the provided settings
#              - write_order: Writes the result of the order processing back to the MySQL database
#              - execute: Main function that orchestrates the reading and writing of orders
# -----------------------------------
# Helper functions
get_env_or_stop <- function(var_name) {
  value <- Sys.getenv(var_name, unset = NA)
  if (is.na(value) || value == "") {
    stop(paste("Missing required environment variable:", var_name))
  }
  return(value)
}

read_settings <- function() {
  args <- commandArgs(trailingOnly = TRUE)
  id <- as.numeric(args[1])
  adjust_dose <- as.logical(args[2])
  error_msg <- args[3]

  if (length(args) < 1) {
    stop("Not enough arguments provided")
  }

  settings <- list(
    mysql_host = get_env_or_stop("R_MYSQL_HOST"),
    mysql_user = get_env_or_stop("R_MYSQL_USER"),
    mysql_password = get_env_or_stop("R_MYSQL_PASSWORD"),
    mysql_db = get_env_or_stop("R_MYSQL_DB"),
    mysql_table = get_env_or_stop("R_MYSQL_TABLE"),
    r_worker = get_env_or_stop("R_WORKER"),
    id = id,
    adjust_dose = adjust_dose,
    error_msg = error_msg
  )
  return(settings)
}

read_order <- function(settings) {
  host_s <- strsplit(settings$mysql_host, ":") |>
    unlist()

  con <- dbConnect(
    RMariaDB::MariaDB(),
    host = host_s[1],
    port = as.numeric(host_s[2]),
    user = settings$mysql_user,
    password = settings$mysql_password,
    dbname = settings$mysql_db
  )

  query <- sprintf("SELECT * FROM `%s` WHERE id = ? LIMIT 1", settings$mysql_table)
  result <- dbGetQuery(con, query, params = list(settings$id))
  dbDisconnect(con)

  if (nrow(result) == 0) {
    stop(sprintf("No order found for ID %d", settings$id))
  }

  if (is.na(result$order_data)) {
    stop(sprintf("Order for ID %d is empty", settings$id))
  }

  order <- list(
    id = result$id,
    order_id = result$order_id,
    order = jsonlite::fromJSON(result$order_data),
    order_data = result$order_data,
    precheck_passed = result$precheck_passed,
    precheck_result = result$precheck_result
  )

  return(order)
}

write_order <- function(settings, success, results_json, pdf_path) {
  host_s <- strsplit(settings$mysql_host, ":") |>
    unlist()

  con <- dbConnect(
    RMariaDB::MariaDB(),
    host = host_s[1],
    port = as.numeric(host_s[2]),
    user = settings$mysql_user,
    password = settings$mysql_password,
    dbname = settings$mysql_db
  )

  encoded_pdf <- base64encode(pdf_path)
  if (is.null(encoded_pdf)) {
    stop(sprintf("Failed to read PDF file: %s", pdf_path))
  }
  # delete_tmp_folder(pdf_path)

  query <- sprintf(
    "UPDATE `%s` SET result_success = ?, result_json = ?, result_pdf = ? WHERE id = ? LIMIT 1",
    settings$mysql_table
  )

  dbExecute(
    con, query,
    params = list(success, results_json, encoded_pdf, settings$id)
  )
  dbDisconnect(con)

  invisible(NULL)
}
