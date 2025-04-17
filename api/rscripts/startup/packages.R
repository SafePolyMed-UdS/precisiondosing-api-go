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
    "stringr", "plotly", "gargoyle", "readxl",
    "future", "promises", "R6", "lubridate", "fs", "configr",
    "jsonlite", "mongolite", "units",
    "cookies", "data.table", "checkmate", "hms", "tictoc",
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
  install.packages("pak")
  pak::pkg_install(packages())
}

install_tinytex <- function() {
  tinytex::install_tinytex()
  tl_packages <- c(
    "fancyhdr", "lipsum", "colortbl",
    "environ", "fp", "pgf", "tcolorbox",
    "trimspaces", "tikzfill", "listings",
    "pdfcol", "listingsutf8"
  )
  tinytex::tlmgr_install(tl_packages)
}
