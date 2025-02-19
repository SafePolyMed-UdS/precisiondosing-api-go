# precisiondosing-api-go

# Endpoints:

Information Endpoints:

- https://doseadjustservice.precisiondosing.de/api/v1/sys/info
- https://doseadjustservice.precisiondosing.de/api/v1/sys/ping

User Login Endpoints:

- https://doseadjustservice.precisiondosing.de/api/v1/user/login
- https://doseadjustservice.precisiondosing.de/api/v1/user/refresh-token

Dose Adjustment Endpoints:

- https://doseadjustservice.precisiondosing.de/api/v1/dose/precheck
- https://doseadjustservice.precisiondosing.de/api/v1/dose/adjust

# Input:

```json
{
  "patient_id": 2,
  "patient_characteristics": {
    "age": 60,
    "weight": 65,
    "height": 168,
    "sex": "female",
    "ethnicity": "asian",
    "kidney_disease": true,
    "liver_disease": true
  },
  "patient_pgx_profile": [
    {
      "gene": "CYP2D6",
      "allele1": "*",
      "allele1_cnv_multiplier": 2,
      "allele2": "*",
      "allele2_cnv_multiplier": 2,
      "phenotype": "Poor metabolizer"
    },
    {
      "gene": "CYP2C19",
      "allele1": "*",
      "allele1_cnv_multiplier": 2,
      "allele2": "*",
      "allele2_cnv_multiplier": 2,
      "phenotype": "Ultrarapid metabolizer"
    },
    {
      "gene": "CYP2C9",
      "allele1": "*1",
      "allele1_cnv_multiplier": 1,
      "allele2": "*3",
      "allele2_cnv_multiplier": 1,
      "phenotype": "Intermediate metabolizer ANDERS"
    },
    {
      "gene": "SLCO1B1",
      "allele1": "521CC",
      "allele1_cnv_multiplier": 2,
      "allele2": "521CC",
      "allele2_cnv_multiplier": 2,
      "phenotype": "n/a"
    }
  ],
  "drugs": [
    {
      "product": {
        "product_name": "Beloc-Zok 95mg",
        "atc": "C07AB02",
        "strength": 95,
        "strength_unit": "milligram"
      },
      "active_substance": ["Metoprolol"],
      "intake_cycle": {
        "starting_at": "2024-11-03",
        "frequency": "days",
        "frequency_modifier": 1,
        "intakes": [
          {
            "cron": "0 8 */1 * *",
            "raw_time_str": "08:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "cron": "0 18 */1 * *",
            "raw_time_str": "18:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "product": {
        "product_name": "Mayzent 2mg",
        "atc": "L04AA42",
        "strength": 2,
        "strength_unit": "milligram"
      },
      "active_substance": ["Siponimod"],
      "intake_cycle": {
        "starting_at": "2024-09-16",
        "frequency": "days",
        "frequency_modifier": 1,
        "intakes": [
          {
            "cron": "0 8 */1 * *",
            "raw_time_str": "08:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "product": {
        "product_name": "Amiodaron 200 Heumann",
        "atc": "C01BD01",
        "strength": 200,
        "strength_unit": "milligram"
      },
      "active_substance": ["Amiodaron"],
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "days",
        "frequency_modifier": 1,
        "intakes": [
          {
            "cron": "0 8 */1 * *",
            "raw_time_str": "08:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "cron": "0 13 */1 * *",
            "raw_time_str": "13:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "cron": "0 18 */1 * *",
            "raw_time_str": "18:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "product": {
        "product_name": "Fevarin 100mg",
        "atc": "N06AB08",
        "strength": 100,
        "strength_unit": "milligram"
      },
      "active_substance": ["Fluvoxamin"],
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "days",
        "frequency_modifier": 1,
        "intakes": [
          {
            "cron": "0 8 */1 * *",
            "raw_time_str": "08:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "product": {
        "product_name": "ESOMEP 20mg",
        "atc": "A02BC05",
        "strength": 20,
        "strength_unit": "milligram"
      },
      "active_substance": ["Esomeprazol"],
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "weeks",
        "frequency_modifier": 1,
        "intakes": [
          {
            "cron": "0 8 */7 * *",
            "raw_time_str": "08:00",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    }
  ]
}
```

# TODO:

- [x] Log to console option
- [ ] User management
- [ ] Input JSON validation
- [ ] JSON structured reading
- [ ] Precheck endpoint
- [ ] Task database
- [ ] Job queue handler
