# -----------------------------------
# Description: Contains helper functions for the main script
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# Notes      : - get_env_or_stop: Retrieves an environment variable or stops execution if not found
#              - read_settings: Reads settings from environment variables and command line arguments
#              - read_order: Reads an order from a MySQL database based on the provided settings
#              - write_order: Writes the result of the order processing back to the MySQL database
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
  model_path <- args[4]

  dbg_out(
    sprintf(
      "Read settings from command line arguments. ID: %s, Adjust Dose: %s, Error Message: %s, Model Path: %s, Workers: %s",
      id, adjust_dose, error_msg, model_path, as.numeric(get_env_or_stop("R_WORKER"))
    )
  )
  if (length(args) < 1) {
    stop("Not enough arguments provided")
  }

  settings <- list(
    mysql_host = get_env_or_stop("R_MYSQL_HOST"),
    mysql_user = get_env_or_stop("R_MYSQL_USER"),
    mysql_password = get_env_or_stop("R_MYSQL_PASSWORD"),
    mysql_db = get_env_or_stop("R_MYSQL_DB"),
    mysql_table = get_env_or_stop("R_MYSQL_TABLE"),
    r_worker = as.numeric(get_env_or_stop("R_WORKER")),
    id = id,
    adjust_dose = adjust_dose,
    model_path = model_path,
    error_msg = error_msg
  )
  return(settings)
}

connect_db <- function(settings) {
  host_s <- strsplit(settings$mysql_host, ":") |>
    unlist()

  host <- host_s[1]
  port <- 3306

  if (length(host_s) == 2) {
    port <- as.numeric(host_s[2])
  }

  con <- dbConnect(
    RMariaDB::MariaDB(),
    host = host,
    port = port,
    user = settings$mysql_user,
    password = settings$mysql_password,
    dbname = settings$mysql_db
  )

  return(con)
}

read_order <- function(settings) {
  con <- connect_db(settings)
  on.exit(dbDisconnect(con), add = TRUE)

  query <- sprintf("SELECT * FROM `%s` WHERE id = ? LIMIT 1", settings$mysql_table)
  result <- dbGetQuery(con, query, params = list(settings$id))

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

write_order <- function(settings, results_json, pdf_path) {
  con <- connect_db(settings)
  on.exit(dbDisconnect(con), add = TRUE)

  if (!file.exists(pdf_path)) {
    stop(sprintf("PDF file does not exist: %s", pdf_path))
  }
  encoded_pdf <- base64encode(pdf_path)
  if (is.null(encoded_pdf)) {
    stop(sprintf("Failed to read PDF file: %s", pdf_path))
  }

  query <- sprintf(
    "UPDATE `%s` SET process_result_pdf = ? WHERE id = ? LIMIT 1",
    settings$mysql_table
  )

  rows_affected <- dbExecute(
    con, query,
    params = list(encoded_pdf, settings$id)
  )
  if (rows_affected != 1) {
    stop(sprintf("Failed to update order with ID %d", settings$id))
  }

  invisible(NULL)
}

dbg_out <- function(...) {
  cat(..., file = stderr())
  cat("\n", file = stderr())
}
