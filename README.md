# SafePolyMed doseadjustservice API

## API Testing with Bruno

This repository uses [Bruno](https://www.usebruno.com/) for API testing. The API request collection is located in the `/bruno` directory.

### Setup

To use Bruno with this project, you need to create a `.env` file in the `/bruno` directory with the following structure.

```bash
LOGIN=YOUR_EMAIL_HERE
PASSWORD="your_password_here"
URL=127.0.0.1:3333

LOGIN_PROD=YOUR_EMAIL_HERE
PASSWORD_PROD="your_production_password_here"
URL_PROD=https://doseadjustservice.precisiondosing.de/
```

### Usage

Make sure to use the appropriate environment in Bruno when running tests:

- **Local environment**: Uses `LOGIN`, `PASSWORD`, and `URL`.
- **Production environment**: Uses `LOGIN_PROD`, `PASSWORD_PROD`, and `URL_PROD`.

Select the correct environment in Bruno before executing API requests.

## Endpoints

Information Endpoints:

- [System Info](https://doseadjustservice.precisiondosing.de/api/v1/sys/info)
- [System Ping](https://doseadjustservice.precisiondosing.de/api/v1/sys/ping)

User Login Endpoints:

- [User Login](https://doseadjustservice.precisiondosing.de/api/v1/user/login)
- [Refresh Token](https://doseadjustservice.precisiondosing.de/api/v1/user/refresh-token)

Dose Adjustment Endpoints:

- [Dose Precheck](https://doseadjustservice.precisiondosing.de/api/v1/dose/precheck)
- [Dose Adjust](https://doseadjustservice.precisiondosing.de/api/v1/dose/adjust)

Model Endpoints:

- [Models](https://doseadjustservice.precisiondosing.de/api/v1/models)

## Input

```json
{
  "patient_id": 2,
  "patient_characteristics": {
    "age": 60,
    "weight": 40,
    "height": 150,
    "sex": "female",
    "ethnicity": "asian",
    "kidney_disease": false,
    "liver_disease": false
  },
  "patient_pgx_profile": [
    {
      "gene": "CYP2D6",
      "allele1": "*1",
      "allele1_cnv_multiplier": 2,
      "allele2": "*2",
      "allele2_cnv_multiplier": 2
    }
  ],
  "drugs": [
    {
      "active_substances": ["Voriconazole"],
      "adjust_dose": true,
      "product": {
        "product_name": "Beloc-Zok 95mg",
        "atc": "C07AB02",
        "strength": 95,
        "strength_unit": "milligram"
      },
      "intake_cycle": {
        "starting_at": "2024-11-03",
        "frequency": "daily",
        "frequency_modifier": 1,
        "intakes": [
          {
            "raw_time_str": "08:00",
            "cron": "0 8 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "raw_time_str": "18:00",
            "cron": "0 18 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "active_substances": ["Imatinib"],
      "adjust_dose": false,
      "product": {
        "product_name": "Amiodaron 200 Heumann",
        "atc": "C01BD01",
        "strength": 200,
        "strength_unit": "milligram"
      },
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "daily",
        "frequency_modifier": 1,
        "intakes": [
          {
            "raw_time_str": "08:00",
            "cron": "0 8 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "raw_time_str": "13:00",
            "cron": "0 13 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          },
          {
            "raw_time_str": "18:00",
            "cron": "0 18 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "active_substances": ["Cimetidine"],
      "adjust_dose": false,
      "product": {
        "product_name": "Fevarin 100mg",
        "atc": "N06AB08",
        "strength": 100,
        "strength_unit": "milligram"
      },
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "daily",
        "frequency_modifier": 1,
        "intakes": [
          {
            "raw_time_str": "08:00",
            "cron": "0 8 */1 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    },
    {
      "active_substances": ["Clopidogrel"],
      "adjust_dose": false,
      "product": {
        "product_name": "ESOMEP 20mg",
        "atc": "A02BC05",
        "strength": 20,
        "strength_unit": "milligram"
      },
      "intake_cycle": {
        "starting_at": "2024-12-01",
        "frequency": "weekly",
        "frequency_modifier": 1,
        "intakes": [
          {
            "raw_time_str": "08:00",
            "cron": "0 8 */7 * *",
            "dosage": 1,
            "dosage_unit": "tablets"
          }
        ]
      }
    }
  ]
}
```

## TODO

### Mandatory

- [ ] Check in MedInfo for synonyms of active substances -> PreCheck
- [x] Logging Framework with implementation in all functions
- [x] Test with real R call with return data
- [ ] Handle failed R calls -> should we even send a response?
- [ ] Real adjustment with R backend
- [ ] Production push with Docker

### Optional

- [ ] Endpoint for job queue overview
- [ ] Polish Swagger documentation
