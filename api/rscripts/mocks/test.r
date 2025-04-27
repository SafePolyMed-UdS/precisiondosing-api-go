# RETURN MUST BE A SERIALIZED JSON WITH THE FOLLOWING STRUCTURE
# {
# "success"    : Boolean,
# "error_msg"  : String with error message if success = FALSE,
# "process_log": String with process log
# }

# -----------------------------------------------------------
# Helpers
# -----------------------------------------------------------
# Use this function to create a JSON Output to standard out
.returnJSON <- function(success, error_msg, process_log) {
  result <- list(
    success = success,
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
      .returnJSON(out$success, out$error_msg, out$process_log)
    },
    error = function(e) {
      .returnJSON(
        success = FALSE,
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
    success = TRUE,
    error_msg = "",
    process_log = paste("Success: Order ID", id, "processed successfully.")
  )
}

# -----------------------------------------------------------
# Execute the function and return the result
# -----------------------------------------------------------
safeReturn(basic_success)
