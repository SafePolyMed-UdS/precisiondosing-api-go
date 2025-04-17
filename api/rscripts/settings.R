# -----------------------------------
# Description: Settings for the DSS-API
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# -----------------------------------
source("startup/settings_helper.R")

create_settings <- function() {
  settings <- list(
    DEBUG_MODE = FALSE,
    DEBUG_CREATE_FAKE = FALSE,
    DEBUG_LOAD_FAKE = FALSE,
    SERVER_OPTIONS = list(
      WORKERS = 4L,
      MULTISESSION = FALSE
    ),
    INSTALL_TINYTEX = TRUE,
    PATHS = list(
      MODELS = .get_pkml_paths("models"),
      REPORTS = "report",
      REPORT_EXAMPLE = "assets/tests/example_report.pdf",
      TEST_DATA = "assets/test_data",
      FAKE_DATA = "assets/fake_data",
      FAKE_SIM = "fake_sim",
      FAKE_MAP = "fake_map",
      MAP_LOGS = "../runs/map_log.csv"
    ),
    DUMMY_MODE = list(
      write = "full",
      read = "full"
    ),
    TOOLNAME = "DDGI DSS Simulation Scripts",
    TOOL_VER = "0.1.0",
    TIMEZONE = "CET",
    DARK_MODE = TRUE,
    REPORT = list(
      markdown_success = file.path("report", "report_success.Rmd"),
      markdown_failed = file.path("report", "report_failed.Rmd"),
      outfile_name = "Report"
    ),
    VALUES = list(
      MODEL_CONFIG = .read_model_definitions("models"),
      POPULATIONS = list(
        "European" = "European_ICRP_2002",
        "Black American" = "BlackAmerican_NHANES_1997",
        "White American" = "WhiteAmerican_NHANES_1997",
        "Asian" = "Asian_Tanaka_1996",
        "Japanese" = "Japanese_Population",
        "Mexican American - White" = "MexicanAmericanWhite_NHANES_1997"
      ),
      SEXES = list(
        "Male" = "MALE",
        "Female" = "FEMALE",
        "Unknown" = "MALE"
      ),
      model_defaults = list(
        SIM_CORES = 5L
      ),
      limits = list(
        max_sim_time = 84, # 3 months
        min_weight = 30,
        max_weight = 250,
        min_height = 120,
        max_height = 220,
        sim_hours_after_last_dose = 30,
        sim_resolution_points_min = 1,
        min_age = 18,
        max_age = 120,
        max_id_length = 20,
        max_doses_per_compound = 20,
        input_max_rows = 20 # clinical data
      )
    )
  )

  return(settings)
}
API_SETTINGS <- create_settings()
