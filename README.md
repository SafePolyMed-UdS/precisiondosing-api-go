# precisiondosing-api-go

## Description
This is a Go API for the SafePolyMed project providing endpoints for the precision dosing service.
The API is built using the Go programming language and the Gin web framework.
It is a RESTful API that provides two endpoints:
- `/precheck` - This endpoint is used to check if a dose adaptation is possible for a given patient and drug.
- `/adapt` - This endpoint is used to adapt a dose for a given patient and drug.

The dose adaptation routine is performed in the `R` programming language using the `ospsuite` package.
