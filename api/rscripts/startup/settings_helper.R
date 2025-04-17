# -----------------------------------
# Description: Helper functions for setting up the environment
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# -----------------------------------
.read_model_definitions <- function(folder) {
  models <- c(character(0))
  for (child in list.dirs(folder, full.names = TRUE, recursive = FALSE)) {
    if (file.exists(file.path(child, "models.yaml"))) {
      models <- c(models, yaml::read_yaml(file.path(child, "models.yaml")))
    }
  }
  return(models)
}

.get_pkml_paths <- function(folder) {
  pkml_paths <- list()
  for (child in list.dirs(folder, full.names = TRUE, recursive = FALSE)) {
    if (file.exists(file.path(child, "pkml"))) {
      model_name <- basename(child) |>
        str_remove_all("shiny-models-") |>
        str_remove_all("[^[:alnum:]]")
      pkml_paths[[model_name]] <- file.path(child, "pkml")
    }
  }
  return(pkml_paths)
}

.get_example_paths <- function(folder) {
  examples <- list()
  for (child in list.dirs(folder, full.names = TRUE, recursive = FALSE)) {
    if (file.exists(file.path(child, "assets/examples/examples.yaml"))) {
      examples_inner <- yaml::read_yaml(file.path(child, "assets/examples/examples.yaml")) |>
        lapply(\(x) {
          list(
            name = x$name,
            file = file.path(child, x$file)
          )
        })

      examples <- c(examples, examples_inner)
    }
  }

  return(examples)
}

.get_template_path <- function(folder) {
  examples <- .get_example_paths(folder)
  template <- examples |>
    purrr::pluck(2) |>
    purrr::pluck("file")
}
