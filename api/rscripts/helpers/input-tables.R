# Input
# -----------------------------------
predictionInputData <- function() {
  data_class <- R6Class(public = list(
    user_data = userInputData(),
    sim_time = NA
  ))

  data_class$new()
}
userInputData <- function() {
  data_class <- R6Class(public = list(
    model_family = NA_character_,
    model = NA_character_,
    available_perpetrators = NA_character_,
    selected_perpetrators = NA_character_,
    doses = inputTableData("Drug", "Date", "Clock time", "Dose"),
    doses_unit = NA_character_,
    genetics = inputGeneticData("Gene", "Genotype"),
    clinical_conc = inputTableData("Drug", "Date", "Clock time", "Concentration"),
    clinical_conc_unit = NA_character_,
    genetic_mapping = NULL,
    kcat_table = NULL,
    map_table = NULL,
    model_id = NA_character_,
    id = NA_character_,
    weight = NA,
    height = NA,
    age = NA,
    sex = NA_character_,
    ethnicity = NA_character_
  ))

  data_class$new()
}

inputGeneticData <- function(gene_name = "Gene",
                             genotype = "Genotype") {
  table <- data.frame(
    C1 = rep(NA_character_, 0),
    C2 = rep(NA_character_, 0)
  ) |> set_names(c(gene_name, genotype))

  data_class <- R6Class(public = list(
    table = table
  ))

  data_class$new()
}

inputTableData <- function(drug_name = "Drug",
                           date_name = "Date",
                           time_name = "Time",
                           value_name = "Value") {
  table <- data.frame(
    C1 = rep(NA_character_, 0),
    C2 = rep(NA_Date_, 0),
    C3 = rep(as.POSIXct(NA, format = "%H:%M"), 0),
    C4 = rep(NA_real_, 0)
  ) |>
    set_names(c(drug_name, date_name, time_name, value_name))

  value_unit <- NA_character_

  data_class <- R6Class(public = list(
    table = table,
    value_unit = value_unit
  ))

  data_class$new()
}

# Output
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
