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
        api_dose_adjustments(order, settings, API_SETTINGS)
      }
    },
    error = function(e) {
      # Reportable error handling
      # -----------------------------------
      if (inherits(e, "report_error")) {
        # create error report
        dbg_out("Creating simulation-error report...")
        report_data <- list(
          order = order,
          errors = list(paste0("Errors during simulation routine: ", e$message))
        )
        tryCatch(
          {
            dbg_out("Creating error report for reportable error: ", e$message)
            report <- render_error_pdf(
              results = report_data,
              api_settings = API_SETTINGS
            )
            dbg_out("Successfully created error report for reportable error...")
            write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
            dbg_out("Successfully saved error report to database...")
          },
          error = function(e) {
            stop(paste("Could not create error report for Order ID", order$order_id, ":", e$message))
          }
        )

        res <- list(
          dose_adjusted = FALSE,
          error = FALSE,
          error_msg = "",
          call_stack = paste0(
            "Error occured during dose adaptation for order ", order$id,
            ". PDF saved to database."
          )
        )
        return(res)
      } else {
        # Non-reportable error handling
        # -----------------------------------
        stop(paste("Unexpected error: ", e$message))
      }
    }
  )

  # Successful simulation
  # -----------------------------------
  if (API_SETTINGS$DEBUG_CREATE_FAKE) {
    sim_results <- saveRDS(sim_results, "mocks/mock_sim_results.rds")
  }

  if (!"error" %in% names(sim_results)) {
    tryCatch(
      {
        dbg_out("Creating success report...")
        report <- render_success_pdf(
          user_data = sim_results$module_data$user_data,
          order = order,
          dose_pk = sim_results$output_data$dose_pk$data,
          api_settings = API_SETTINGS
        )
        write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
      },
      error = function(e) {
        stop(paste0("Could not create report for Order ID ", order$order_id, ": ", e$message))
      }
    )

    process_log <- paste(
      "Order ID", order$order_id, "processed successfully. PDF saved to database."
    )

    res <- list(
      dose_adjusted = TRUE,
      call_stack = list(process_log),
      error = FALSE,
      error_msg = ""
    )
  } else {
    res <- sim_results
  }
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
      dbg_out("Successfully created error report for order precheck...")
      write_order(settings, jsonlite::toJSON(order$order_data, auto_unbox = TRUE), report)
      dbg_out("Successfully saved error report to database...")
    },
    error = function(e) {
      stop(paste("Could not create error report for Order ID", order$order_id, ":", e$message))
    }
  )
  res <- list(
    dose_adjusted = FALSE,
    error = FALSE,
    error_msg = "",
    call_stack = list(paste("Success: Order ID", order$order_id, "processed successfully."))
  )
  return(res)
}
