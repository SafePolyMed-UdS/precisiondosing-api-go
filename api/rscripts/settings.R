# -----------------------------------
# Description: Settings for the DSS-API
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# -----------------------------------
source("startup/settings_helper.R")

create_settings <- function(model_path) {
  settings <- list(
    DEBUG_CREATE_FAKE = FALSE,
    DEBUG_LOAD_FAKE = FALSE,
    SERVER_OPTIONS = list(
      WORKERS = 4L,
      MULTISESSION = FALSE
    ),
    INSTALL_TINYTEX = TRUE,
    PATHS = list(
      MODELS = .get_pkml_paths(model_path),
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
      markdown_success = file.path("report_success.Rmd"),
      markdown_failed = file.path("report_failed.Rmd"),
      outfile_name = "Report"
    ),
    VALUES = list(
      MODEL_CONFIG = .read_model_definitions(model_path),
      POPULATIONS = list(
        "european" = "European_ICRP_2002",
        "white american" = "WhiteAmerican_NHANES_1997",
        "black american" = "BlackAmerican_NHANES_1997",
        "mexican" = "MexicanAmericanWhite_NHANES_1997",
        "asian" = "Asian_Tanaka_1996",
        "japanese" = "Japanese_Population",
        "other" = "European_ICRP_2002",
        "white" = "European_ICRP_2002",
        "african" = "BlackAmerican_NHANES_1997",
        "other_ethnicity" = "European_ICRP_2002",
        "mixed_background" = "European_ICRP_2002",
        "unknown" = "European_ICRP_2002"
      ),
      SEXES = list(
        "male" = "MALE",
        "female" = "FEMALE",
        "unknown" = "MALE"
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
