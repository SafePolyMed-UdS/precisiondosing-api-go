# -----------------------------------
# Description: Helper functions for reading environment variables
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# Notes      : - Reads environment variables from the system or .env file
# -----------------------------------
.env_to_bool <- function(x, var_name) {
  x <- tolower(x)
  if (x == "true") {
    return(TRUE)
  } else if (x == "false") {
    return(FALSE)
  }
  stop(paste0("Environment variable`", var_name, "` must be `true` or `false`."),
    call. = FALSE
  )
}

.env_to_numeric <- function(x, var_name) {
  x <- as.numeric(x) |> suppressWarnings()
  if (is.na(x)) {
    stop(paste0("Environment variable `", var_name, "` must be numeric."),
      call. = FALSE
    )
  }
  return(x)
}

.env_vars <- function(setted_vars) {
  vars <- list(
    mongo_host = Sys.getenv("MONGO_HOST", unset = NA),
    mongo_user = Sys.getenv("MONGO_USER", unset = NA),
    mongo_pass = Sys.getenv("MONGO_PASS", unset = NA),
    mongo_port = Sys.getenv("MONGO_PORT", unset = NA),
    debug_mode = Sys.getenv("DEBUG_MODE", unset = NA),
    create_fake_data = Sys.getenv("CREATE_FAKE_DATA", unset = NA),
    load_fake_data = Sys.getenv("LOAD_FAKE_DATA", unset = NA),
    multisession = Sys.getenv("R_MULTISESSION", unset = NA),
    workers = Sys.getenv("R_WORKERS", unset = NA)
  )

  setted_vars <- purrr::keep(setted_vars, \(x) !is.na(x))
  protect_names <- names(setted_vars)

  vars <- vars[setdiff(names(vars), protect_names)]
  vars <- c(vars, setted_vars)
  return(vars)
}

read_env <- function() {
  vars <- .env_vars(list())

  if (purrr::map_lgl(vars, is.na) |> any()) {
    if (readRenviron(".env")) {
      vars <- .env_vars(vars)

      if (purrr::map_lgl(vars, is.na) |> any()) {
        stop("Environment variables missing reading from .env file.")
      }
      message("Read some enviroment variables from .env file!")
    } else {
      stop("Environment variables missing and no .env file found.")
    }
  }

  # casting
  vars$multisession <- .env_to_bool(vars$multisession, "R_MULTISESSION")
  vars$mongo_port <- .env_to_numeric(vars$mongo_port, "MONGO_PORT")
  vars$workers <- .env_to_numeric(vars$workers, "R_WORKERS")
  vars$debug_mode <- .env_to_bool(vars$debug_mode, "DEBUG_MODE")
  vars$create_fake_data <- .env_to_bool(vars$create_fake_data, "CREATE_FAKE_DATA")
  vars$load_fake_data <- .env_to_bool(vars$load_fake_data, "LOAD_FAKE_DATA")

  return(vars)
}
