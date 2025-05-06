# -----------------------------------
# Description: Script rendering
# Author     : Dominik Selzer
# Date       : 2025-04-17
# Notes      : - `render_pdf` function to render the report
#              - `render_error_pdf` function to render the report in case of an error
#              - `render_success_pdf` function to render the report in case of success
# -----------------------------------
render_pdf <- function(markdown_template, outfile_name, params) {
  # copy folder report to tmp_dir
  markdown_folder <- tempdir()
  file.copy(
    from = list.files("report", full.names = TRUE),
    to = file.path(markdown_folder),
    overwrite = TRUE
  )

  markdown_dest <- file.path(markdown_folder, markdown_template)

  tryCatch(
    {
      result <- rmarkdown::render(
        input = markdown_dest,
        output_file = outfile_name,
        output_dir = markdown_folder,
        params = params,
        quiet = TRUE,
        envir = new.env(parent = globalenv())
      ) |>
        suppressWarnings()
    },
    error = function(e) {
      logfile <- file.path(markdown_folder, "report_success.log")
      if (file.exists(logfile)) {
        log_content <- paste(readLines(logfile), collapse = "\n")
        stop(paste("Error rendering PDF with log:", log_content))
      }
      stop(paste("Error rendering PDF:", e$message))
    }
  )

  return(result)
}

render_error_pdf <- function(results, api_settings) {
  params <- results

  pdf_path <- render_pdf(
    markdown_template = api_settings$REPORT$markdown_failed,
    outfile_name = file.path(
      api_settings$PATHS$REPORTS,
      paste0(api_settings$REPORT$outfile_name, "_failed.pdf")
    ),
    params = params
  )

  return(pdf_path)
}

render_success_pdf <- function(user_data, output_data, dose_pk, order, api_settings) {
  params <- list(
    user_data = user_data,
    dose_pk = dose_pk,
    order = order,
    settings = api_settings
  )

  pdf_path <- render_pdf(
    markdown_template = api_settings$REPORT$markdown_success,
    outfile_name = file.path(
      api_settings$PATHS$REPORTS,
      paste0(api_settings$REPORT$outfile_name, "_success.pdf")
    ),
    params = params
  )

  return(pdf_path)
}

delete_tmp_folder <- function(pdf_path) {
  # Extract the temporary directory from the PDF path
  temp_dir <- dirname(pdf_path)

  # Delete the temporary directory and its contents
  if (dir_exists(temp_dir)) {
    dir_delete(temp_dir)
  }
}
