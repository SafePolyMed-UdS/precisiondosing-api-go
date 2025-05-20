display_population <- function(population) {
  map_pop <- list(
    "european" = "European",
    "white american" = "White American",
    "black american" = "Black American",
    "mexican" = "Mexican American",
    "asian" = "Asian",
    "japanese" = "Japanese",
    "other" = "Other",
    "white" = "White European",
    "african" = "African",
    "other_ethnicity" = "Other",
    "mixed_background" = "Mixed Background",
    "unknown" = "Unknown"
  )

  if (!(population %in% names(map_pop))) {
    error <- paste0("In display_population: population '", population, "' not found in map_pop.")
    stop(error)
  }
  return(map_pop[[population]])
}
