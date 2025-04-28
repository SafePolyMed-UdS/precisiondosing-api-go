# -----------------------------------------------------------
# Helpers
# -----------------------------------------------------------
# Use this function to create a JSON Output to standard out
.returnJSON <- function(dose_adjusted, process_log, error, error_msg, call_stack = character(0)) {
  result <- list(
    dose_adjusted = jsonlite::unbox(dose_adjusted),
    error = jsonlite::unbox(error),
    error_msg = jsonlite::unbox(error_msg),
    process_log = jsonlite::unbox(process_log)
  )

  if (length(call_stack) > 0) {
    result$call_stack <- call_stack
  }

  json_str <- suppressMessages(suppressWarnings({
    jsonlite::toJSON(result, auto_unbox = FALSE)
  }))
  cat(json_str)
  cat("\n")
}

# This function is used to suppress all output from a function
.reallySilent <- function(expr) {
  tf <- tempfile()
  sink(tf)
  on.exit(
    {
      sink()
      suppressWarnings(try(unlink(tf), silent = TRUE))
    },
    add = TRUE
  )

  result <- suppressMessages(suppressWarnings(
    force(expr)()
  ))

  invisible(result)
}

# this should eat up all possible output from the a function
# and send the result to the console
safeReturn <- function(fun) {
  tryCatch(
    {
      out <- .reallySilent(fun)
      .returnJSON(out$dose_adjustment, out$process_log, out$error, out$error_msg)
    },
    error = function(e) {
      calls <- sys.calls()
      filtered_calls <- calls[!grepl("tryCatch|withCallingHandlers", sapply(calls, deparse))]
      call_stack <- unname(
        sapply(filtered_calls, function(call) paste(deparse(call), collapse = " "))
      )

      .returnJSON(
        dose_adjusted = FALSE,
        error = TRUE,
        error_msg = e$message,
        call_stack = call_stack,
        process_log = ""
      )
    }
  )
}

# -----------------------------------------------------------
# Main functions
# -----------------------------------------------------------
basic_success <- function() {
  library(DBI)
  library(RMariaDB)
  library(base64enc)
  library(jsonlite)
  cmdArgs <- commandArgs(trailingOnly = TRUE)
  if (length(cmdArgs) != 3) {
    stop("need 3 commandline arguments")
  }

  id <- cmdArgs[1] |> as.numeric()
  mysql_host <- Sys.getenv("R_MYSQL_HOST") |>
    strsplit(":") |>
    unlist()
  mysql_user <- Sys.getenv("R_MYSQL_USER")
  mysql_password <- Sys.getenv("R_MYSQL_PASSWORD")
  mysql_db <- Sys.getenv("R_MYSQL_DB")
  mysql_table <- Sys.getenv("R_MYSQL_TABLE")

  # Simulate the order processing
  Sys.sleep(1)

  # DB
  con <- dbConnect(
    RMariaDB::MariaDB(),
    host = mysql_host[1],
    port = as.numeric(mysql_host[2]),
    user = mysql_user,
    password = mysql_password,
    dbname = mysql_db
  )
  on.exit(dbDisconnect(con), add = TRUE)

  encoded_pdf <- base64encode("test.pdf")
  if (is.null(encoded_pdf)) {
    stop("Failed to read PDF file")
  }

  query <- sprintf("UPDATE `%s` SET process_result_pdf = ? WHERE id = ? LIMIT 1", mysql_table)

  a <- dbExecute(
    con, query,
    params = list(encoded_pdf, id)
  )

  process_log <- sprintf(
    "Order %d processed successfully. PDF saved to database. %d rows updated.",
    id, a
  )

  list(
    dose_adjustment = TRUE,
    process_log = process_log,
    error = FALSE,
    error_msg = ""
  )
}

# -----------------------------------------------------------
# Execute the function and return the result
# -----------------------------------------------------------
safeReturn(basic_success)
