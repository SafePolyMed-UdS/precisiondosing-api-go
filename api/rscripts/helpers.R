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
  id <- commandArgs(trailingOnly = TRUE) |>
    as.numeric()
  if (length(id) < 1) {
    stop("Not enough arguments provided")
  }

  settings <- list(
    mysql_host = get_env_or_stop("R_MYSQL_HOST"),
    mysql_user = get_env_or_stop("R_MYSQL_USER"),
    mysql_password = get_env_or_stop("R_MYSQL_PASSWORD"),
    mysql_db = get_env_or_stop("R_MYSQL_DB"),
    mysql_table = get_env_or_stop("R_MYSQL_TABLE"),
    id = id
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
  on.exit(dbDisconnect(con), add = TRUE)

  query <- sprintf("SELECT * FROM `%s` WHERE id = ? LIMIT 1", settings$mysql_table)
  result <- dbGetQuery(con, query, params = list(settings$id))

  if (nrow(result) == 0) {
    stop(sprintf("No order found for ID %d", settings$id))
  }

  if (is.na(result$order)) {
    stop(sprintf("Order for ID %d is empty", settings$id))
  }

  list(
    id = result$id,
    order_id = result$order_id,
    order = jsonlite::fromJSON(result$order)
  )
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
  on.exit(dbDisconnect(con), add = TRUE)

  encoded_pdf <- base64encode(pdf_path)
  if (is.null(encoded_pdf)) {
    stop(sprintf("Failed to read PDF file: %s", pdf_path))
  }

  query <- sprintf(
    "UPDATE `%s` SET result_success = ?, result_json = ?, result_pdf = ? WHERE id = ? LIMIT 1",
    settings$mysql_table
  )

  dbExecute(
    con, query,
    params = list(success, results_json, encoded_pdf, settings$id)
  )

  invisible(NULL)
}

execute <- function() {
  tryCatch(
    {
      settings <- read_settings()
      order <- read_order(settings)

      res <- list(
        success = TRUE,
        created_pdf = TRUE,
        msg_user = "Success",
        msg_system = jsonlite::toJSON(order$order, auto_unbox = TRUE)
      )

      res
    },
    error = function(e) {
      list(
        success = FALSE,
        created_pdf = FALSE,
        msg_user = "",
        msg_system = e$message
      )
    }
  )
}
