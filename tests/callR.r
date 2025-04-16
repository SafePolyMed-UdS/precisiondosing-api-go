suppressMessages(suppressWarnings({
  library(jsonlite)

  result <- list(success = TRUE, data = c(1, 2, 3))

  invisible(capture.output({
    # DOIT
  }))

  cat(toJSON(result, auto_unbox = TRUE))
}))
