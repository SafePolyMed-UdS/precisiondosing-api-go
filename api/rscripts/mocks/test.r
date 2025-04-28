# -----------------------------------------------------------
# Helpers
# -----------------------------------------------------------
# Use this function to create a JSON Output to standard out
.returnJSON <- function(dose_adjustment, error, error_msg, process_log) {
  result <- list(
    dose_adjustment = dose_adjustment,
    error = error,
    error_msg = error_msg,
    process_log = process_log
  )

  json_str <- suppressMessages(suppressWarnings({
    jsonlite::toJSON(result, auto_unbox = TRUE)
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
      if (file.exists(tf)) file.remove(tf)
    },
    add = TRUE
  )

  result <- suppressMessages(suppressWarnings(
    force(expr)
  ))

  invisible(result)
}

# this should eat up all possible output from the a function
# and send the result to the console
safeReturn <- function(fun) {
  tryCatch(
    {
      out <- .reallySilent(fun())
      .returnJSON(out$dose_adjustment, out$error, out$error_msg, out$process_log)
    },
    error = function(e) {
      .returnJSON(
        dose_adjustment = FALSE,
        error = TRUE,
        error_msg = e$message,
        process_log = ""
      )
    }
  )
}

# -----------------------------------------------------------
# Main functions
# -----------------------------------------------------------
basic_success <- function() {
  Sys.sleep(5)
  id <- commandArgs(trailingOnly = TRUE) |>
    as.numeric()

  if (length(id) < 1) {
    stop("need orderID as script argument")
  }

  list(
    dose_adjustment = TRUE,
    error = FALSE,
    error_msg = "",
    process_log = paste("Success: Order ID", id, "processed successfully.")
  )
}

# -----------------------------------------------------------
# Execute the function and return the result
# -----------------------------------------------------------
safeReturn(basic_success)
