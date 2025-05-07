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
execute <- function(order, settings, API_SETTINGS) {
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
            "No dose adjustment possible due to errors.",
            e$message
          )
        )
        report <- render_error_pdf(
          results = report_data,
          api_settings = API_SETTINGS
        )
        write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)

        process_log <- paste(
          "Error occured during dose adaptation for order", order$id,
          ". PDF saved to database."
        )

        list(
          dose_adjusted = FALSE,
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
      write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
    },
    error = function(e) {
      stop(paste("Could not create report for Order ID", order$order_id, ":", e$message))
    }
  )

  process_log <- paste(
    "Order ID", order$order_id, "processed successfully. PDF saved to database."
  )
  dbg_out(process_log)

  res <- list(
    dose_adjusted = TRUE,
    process_log = process_log,
    error = FALSE,
    error_msg = ""
  )
  return(res)
}

precheck_error_pdf <- function(order, settings, API_SETTINGS) {
  precheck_results <- list(
    order = order,
    errors = list(
      paste("No dose adjustment possible due to errors in the order precheck.", settings$error_msg)
    )
  )
  tryCatch(
    {
      dbg_out("Creating error report for order precheck...")
      report <- render_error_pdf(
        results = precheck_results,
        api_settings = API_SETTINGS
      )
      dbg_out("Successfully created error report for order precheck.")
      write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
      dbg_out("Successfully saved error report to database.")
    },
    error = function(e) {
      stop(paste("Could not create error report for Order ID", order$order_id, ":", e$message))
    }
  )
  res <- list(
    dose_adjusted = FALSE,
    error = FALSE,
    error_msg = "",
    process_log = paste("Success: Order ID", order$order_id, "processed successfully.")
  )
  dbg_out(toJSON(res, auto_unbox = TRUE))
  return(res)
}
