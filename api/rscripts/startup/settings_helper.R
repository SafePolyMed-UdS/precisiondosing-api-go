# -----------------------------------
# Description: Helper functions for setting up the environment
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# -----------------------------------
.read_model_definitions <- function(folder) {
  dbg_out("Reading model definitions from folder: ", folder)
  models <- c(character(0))
  for (child in list.dirs(folder, full.names = TRUE, recursive = FALSE)) {
    if (file.exists(file.path(child, "models.yaml"))) {
      child_model <- yaml::read_yaml(file.path(child, "models.yaml"))

      models_contained <- child_model[[1]]$models |>
        lapply(\(x) {
          x$pkml_path <- file.path(child, "pkml")
          return(x)
        })
      child_model[[1]]$models <- models_contained

      models <- c(models, child_model)
    }
  }
  return(models)
}

.get_pkml_paths <- function(folder) {
  dbg_out("Getting pkml paths from folder: ", folder)
  pkml_paths <- list()
  for (child in list.dirs(folder, full.names = TRUE, recursive = FALSE)) {
    if (file.exists(file.path(child, "pkml"))) {
      model_name <- basename(child) |>
        stringr::str_remove_all("shiny-models-") |>
        stringr::str_remove_all("[^[:alnum:]]")
      pkml_paths[[model_name]] <- file.path(child, "pkml")
    } else {
      stop(paste0("No pkml folder found in ", child))
    }
  }
  return(pkml_paths)
}
