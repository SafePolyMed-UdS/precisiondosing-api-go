# -----------------------------------
# Description: Test script for the API
# Author     : Dominik Selzer
# Date       : 2025-04-28
# -----------------------------------

# Output definition
# -----------------------------------
# list(
#   dose_adjusted = TRUE/FALSE, - TRUE if dose adjusted was performed, FALSE otherwise
#   error = TRUE/FALSE,           - TRUE if an error occurred, FALSE otherwise
#   error_msg = "error message",  - Error message if an error occurred
#   process_log = "log message"   - Log message for the process
# )

# -----------------------------------------------------------
# Main functions
# -----------------------------------------------------------
execute <- function(order, settings) {
  sim_results <- tryCatch(
    {
      if (API_SETTINGS$DEBUG_LOAD_FAKE) {
        readRDS("mocks/mock_sim_results.rds")
      } else {
        api_dose_adjustments(order, settings)
      }
    },
    error = function(e) {
      if (inherits(e, "report_error")) {
        # create error report
        report_data <- list(
          order = order,
          errors = list(
            "No dose adjusted possible",
            "due to errors \n Traceback: ",
            e$message
          )
        )
        report <- render_error_pdf(
          results = report_data,
          api_settings = API_SETTINGS
        )
        write_order(settings, TRUE, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)

        process_log <- paste(
          "Error occured during dose adaptation for order", order$id,
          ". PDF saved to database."
        )

        list(
          dose_adjustment = FALSE,
          process_log = process_log,
          error = FALSE,
          error_msg = ""
        )
      } else {
        stop(paste("Unexpected error: ", e$message))
      }
    }
  )
  if (API_SETTINGS$DEBUG_CREATE_FAKE) {
    sim_results <- saveRDS(sim_results, "mocks/mock_sim_results.rds")
  }

  tryCatch(
    {
      report <- render_success_pdf(
        user_data = sim_results$module_data$user_data,
        order = order,
        dose_pk = sim_results$output_data$dose_pk$data,
        api_settings = API_SETTINGS
      )
      write_order(settings, TRUE, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
    },
    error = function(e) {
      stop(paste("Could not create report for Order ID", order$order_id, ":", e$message))
    }
  )

  process_log <- paste(
    "Order ID", order$order_id, "processed successfully. PDF saved to database."
  )

  list(
    dose_adjustment = TRUE,
    process_log = process_log,
    error = FALSE,
    error_msg = ""
  )
}

precheck_error_pdf <- function(order, settings) {
  precheck_results <- list(
    order = order,
    errors = list(
      "No dose adjusted possible",
      "due to errors in the order precheck\n Traceback: ",
      settings$error_msg
    )
  )
  tryCatch(
    {
      report <- render_error_pdf(
        results = precheck_results,
        api_settings = API_SETTINGS
      )

      write_order(settings, TRUE, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)

      res <- list(
        dose_adjusted = FALSE,
        error = FALSE,
        error_msg = "",
        process_log = paste("Success: Order ID", order$order_id, "processed successfully.")
      )
    },
    error = function(e) {
      res <- list(
        dose_adjusted = FALSE,
        error = TRUE,
        error_msg = paste("Could not create error report for Order ID", order$order_id, ":", e$message),
        process_log = ""
      )
    }
  )
  return(res)
}
