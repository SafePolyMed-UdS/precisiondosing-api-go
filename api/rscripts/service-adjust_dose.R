# -----------------------------------
# Description: Perform the main simulation task for the service
# Author     : Simeon Rüdesheim
# Date       : 2025-04-17
# -----------------------------------

# Dependencies
# -----------------------------------
source("pksim-sim-engine/src/dose_adaptation_helpers.R")
source("pksim-sim-engine/src/dose_adaptations.R")
source("pksim-sim-engine/src/dose_helpers.R")
source("pksim-sim-engine/src/map.R")
source("pksim-sim-engine/src/model_getters.R")
source("pksim-sim-engine/src/model_helpers.R")
source("pksim-sim-engine/src/model_setters.R")
source("pksim-sim-engine/src/sim_post_processing.R")
source("pksim-sim-engine/src/sim_pre_processing.R")
source("pksim-sim-engine/src/sim_routines.R")
source("pksim-sim-engine/src/sim_routines_helpers.R")
source("pksim-sim-engine/src/simulate.R")
source("pksim-sim-engine/src/units.R")

# Set up output data
# -----------------------------------
predictionPlotData <- function(name) {
  data_class <- R6Class(public = list(
    name = name,
    plot = NULL,
    data = NULL,
    clinical_data = NULL,
    xaxis_scale = "dates",
    yaxis_scale = "lin"
  ))

  data_class$new()
}

predictionOutputData <- function() {
  data_class <- R6Class(public = list(
    simulation_pk = predictionPlotData("pk_simulation"),
    map_pk = predictionPlotData("map_simulation"),
    dose_pk = predictionPlotData("dose_simulation"),
    map_state = NULL
  ))

  data_class$new()
}

# Code
# -----------------------------------
adjust_dose <- function(order) {
  results <- list()

  return(results)
}
