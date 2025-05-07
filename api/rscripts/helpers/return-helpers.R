# -----------------------------------
# Description: Handle output for the API
# Author     : Dominik Selzer
# Date       : 2025-04-28
# -----------------------------------

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
      out <- fun()
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
