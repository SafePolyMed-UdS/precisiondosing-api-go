# -----------------------------------
# Description: Main script for adjusting doses
# Author     : Dominik Selzer
# Date       : 2025-04-17
# Notes      : - Calls the helper functions to read and write orders
#              - Loads libraries and sets up the environment
# -----------------------------------
suppressMessages(suppressWarnings({
  source("startup/packages.R")
  source("helpers.R")
  source("settings.R")

  load_packages()

  invisible({
    result <- execute()
  })

  # rawdog to stdout
  cat(toJSON(result, auto_unbox = TRUE))
}))
