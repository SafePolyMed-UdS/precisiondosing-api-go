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
source("helpers/input-tables.R")

# Main function
# -----------------------------------
api_dose_adjustments <- function(order, settings, API_SETTINGS) {
  payload <- fromJSON(order$precheck_result)
  payload_order <- fromJSON(order$order_data)

  module_data <- predictionInputData()

  # Model info
  # -----------------------------------
  model_id <- payload$model_id
  model_info <- api_get_model_from_id(API_SETTINGS, model_id)
  victim_info <- get_compound_infos(API_SETTINGS, model_info) |>
    pluck(1)
  dbg_out("Loaded model info for model_id: ", model_id, " ...")
  model_compounds <- c(model_info$victim, model_info$perpetrators)

  module_data$user_data$model_family <- model_info$victim
  module_data$user_data$selected_perpetrators <- model_info$perpetrators

  # Virtual Individual
  # -----------------------------------
  individual <- payload$virtual_individual

  # Compounds
  # -----------------------------------
  compounds <- payload$compounds

  compounds_parsed <- list()
  # TODO: MOVE TO NAME_IN_MODEL
  for (co in unique(compounds$name)) {
    co_data <- compounds |>
      filter(name == co)

    co_data_no_sched <- co_data |>
      select(-schedule)

    schedule <- co_data |>
      pull(schedule) |>
      pluck(1) |>
      mutate(name = co) |>
      left_join(
        co_data_no_sched,
        by = "name"
      )
    compounds_parsed[[co]] <- schedule
  }

  dosing_table <- bind_rows(compounds_parsed) |>
    filter(name_in_model %in% model_compounds) |>
    mutate(time = lubridate::ymd_hm(time_str)) |>
    arrange(time) |>
    group_by(name_in_model) |>
    mutate(index = row_number()) |>
    as.data.frame()

  # first occurrence of index 10 = cutoff
  max_time <- dosing_table |>
    filter(index == 10) |>
    first() |>
    pull(time)

  dosing_table_truncated <- dosing_table |>
    filter(time <= max_time) |>
    mutate(Date = as.Date(time)) |>
    mutate(`Clock time` = format(time, "%H:%M")) |>
    # add seconds to Clock time
    mutate(`Clock time` = paste0(`Clock time`, ":00")) |>
    mutate(Dose = dose_amount * dosage) |>
    # FIXME: This unlikely to be robust
    mutate(Drug = str_to_title(name_in_model)) |>
    select(Drug, Date, `Clock time`, Dose)

  module_data$user_data$doses$table <- dosing_table_truncated
  # FIXME: Read from input? Has to be normalized in dosing table
  module_data$user_data$doses$value_unit <- victim_info$dosing_unit


  # Interactions
  # -----------------------------------
  # TODO: Flag modeled interactions
  if (!is.null(payload$interactions)) {
    interactions <- payload$interactions |>
      tidyr::replace_na(list(
        frequency = "unknown",
        relevance = "unknown",
        credibility = "unknown",
        plausibility = "unknown"
      )) |>
      mutate(victim = str_to_title(compounds_left)) |>
      mutate(perpetrator = str_to_title(compounds_right)) |>
      select(victim, perpetrator, frequency, relevance, credibility, plausibility)
  } else {
    interactions <- NULL
  }
  module_data$user_data$interactions <- interactions

  # Clinical data
  # -----------------------------------
  module_data$user_data$clinical_conc$value_unit <- victim_info$std_measurement_unit

  # Patient Characteristics
  # -----------------------------------
  patient_characteristics <- payload_order$patient_characteristics
  module_data$user_data$id <- payload_order$patient_id
  module_data$user_data$age <- patient_characteristics$age
  module_data$user_data$weight <- patient_characteristics$weight
  module_data$user_data$height <- patient_characteristics$height

  module_data$user_data$sex <- patient_characteristics$sex
  module_data$user_data$ethnicity <- patient_characteristics$ethnicity

  # Genetics
  # -----------------------------------
  patient_genetics <- payload_order$patient_pgx_profile |>
    # Concat Genotype
    mutate_at(vars(allele1, allele2), ~ str_replace_all(., "/", "|")) |>
    mutate(Genotype = paste0(allele1_cnv_multiplier, "x", allele1, "/", allele2_cnv_multiplier, "x", allele2)) |>
    mutate(Genotype = str_remove_all(Genotype, "1x")) |> # remove 1x
    # Rename gene to Gene
    rename(Gene = gene)

  genetic_mapping <- get_model_genetics(API_SETTINGS, model_info)

  relevant_genetics <- patient_genetics |>
    filter(Gene %in% genetic_mapping$phenotype_mapping$gene_name) |>
    select(Gene, Genotype)

  module_data$user_data$genetics$table <- relevant_genetics
  module_data$user_data$genetic_mapping <- genetic_mapping
  module_data$user_data$genetics_raw <- patient_genetics
  module_data$user_data$kcat_table <- get_param_table(API_SETTINGS, model_info)


  # Simulate stuff
  # -----------------------------------
  output_data <- predictionOutputData()

  dose_sim_res <- dose_sim_routine_api(
    module_data = module_data,
    output_data = output_data,
    settings = API_SETTINGS,
    individual = individual,
    use_fake_data = FALSE,
    create_fake_data = FALSE
  )

  output_data$dose_pk$data <- dose_sim_res$dose_data

  return(list(
    module_data = module_data,
    output_data = output_data
  ))
}
