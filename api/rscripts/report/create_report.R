# -----------------------------------
# Description: Script rendering
# Author     : Dominik Selzer
# Date       : 2025-04-17
# Notes      : - `render_pdf` function to render the report
#              - `render_error_pdf` function to render the report in case of an error
#              - `render_success_pdf` function to render the report in case of success
# -----------------------------------
render_pdf <- function(markdown_path, outfile_name, params) {
  # create temporary dir
  tmp_dir <- paste0(sample(LETTERS, 10, replace = TRUE), collapse = "")
  dir_create(tmp_dir)

  tryCatch(
    {
      # copy folder report to tmp_dir
      markdown_folder <- file.path(tmp_dir, "report")
      dir_create(markdown_folder)
      file.copy(
        from = file.path("report"),
        to = file.path(tmp_dir),
        recursive = TRUE
      )
      markdown_dest <- file.path(markdown_folder, basename(markdown_path))

      rmarkdown::render(
        input = markdown_dest,
        output_file = outfile_name,
        output_dir = tmp_dir,
        params = params,
        quiet = TRUE,
        envir = new.env(parent = globalenv())
      ) |>
        suppressWarnings()

      return(
        file.path(
          tmp_dir,
          paste0(
            tools::file_path_sans_ext(basename(outfile_name)),
            ".pdf"
          )
        )
      )
    },
    error = function(error) {
      try(dir_delete(tmp_dir))
      stop(error$message, call. = FALSE)
    }
  )
}

render_error_pdf <- function(results, api_settings) {
  params <- results

  pdf_path <- render_pdf(
    markdown_path = api_settings$REPORT$markdown_failed,
    outfile_name = file.path(
      api_settings$PATHS$REPORTS,
      paste0(api_settings$REPORT$outfile_name, "_failed.pdf")
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
