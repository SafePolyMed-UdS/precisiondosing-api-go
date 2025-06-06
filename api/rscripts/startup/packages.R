# -----------------------------------
# Description: Load libraries for the main scripts
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# Notes      : - Loads CRAN packages
#              - Loads remote packages
# -----------------------------------
packages <- function() {
  c(
    "PKNCA", "DT", "dplyr", "tidyr", "purrr", "glue",
    "stringr", "readxl", "R6", "lubridate", "fs", "configr",
    "jsonlite", "units", "data.table", "checkmate", "hms", "tictoc",
    "DBI", "RMariaDB", "base64enc", "rmarkdown", "tinytex",
    "bookdown", "kableExtra", "gt", "viridis", "paletteer"
  )
}

load_packages <- function() {
  suppressPackageStartupMessages({
    lapply(packages(), library, character.only = TRUE)
    remote_packages() |>
      lapply(\(x) library(x$pkg, character.only = TRUE))
  }) |>
    suppressWarnings()
}

remote_packages <- function() {
  x <- list(
    list(
      pkg = "ospsuite.utils",
      repo = "Open-Systems-Pharmacology/OSPSuite.RUtils"
    ),
    list(
      pkg = "tlf",
      repo = "Open-Systems-Pharmacology/TLF-Library"
    ),
    list(
      pkg = "rSharp",
      repo = "Open-Systems-Pharmacology/rSharp"
    ),
    list(
      pkg = "ospsuite",
      repo = "Open-Systems-Pharmacology/OSPSuite-R"
    )
  )
  return(x)
}

install_packages <- function() {
  if (!requireNamespace("pak", quietly = TRUE)) install.packages("pak")
  pak::pkg_install(packages())
  install_tinytex()
}

install_tinytex <- function() {
  tinytex::install_tinytex(force = TRUE)
  tl_packages <- c(
    "fancyhdr", "lipsum", "colortbl",
    "environ", "fp", "pgf", "tcolorbox",
    "trimspaces", "tikzfill", "listings",
    "pdfcol", "listingsutf8", "bookmark"
  )
  tinytex::tlmgr_install(tl_packages)
}
