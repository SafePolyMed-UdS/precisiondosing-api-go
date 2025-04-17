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
  print(tmp_dir)

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
      Sys.sleep(1)
      browser()

      rmarkdown::render(
        input = markdown_dest,
        output_file = outfile_name,
        output_dir = tmp_dir,
        params = params,
        quiet = TRUE,
        envir = new.env(parent = globalenv())
      )
    },
    error = function(error) {
      try(dir_delete(tmp_dir))
      stop(error$message, call. = FALSE)
    }
  )
}

render_error_pdf <- function(results, settings) {
  params <- results

  report <- render_pdf(
    markdown_path = settings$REPORT$markdown_failed,
    outfile_name = file.path(settings$PATHS$REPORTS, paste0(settings$REPORT$outfile_name, "_failed.pdf")),
    params = params
  )
}
