set windows-shell := ["pwsh.exe", "-NoLogo", "-c"]

r := if os_family() == "windows" {"RScript.exe"} else {"RScript"}

_default:
    @ just -l

# Runs the API
[group('dev')]
run:
    @ cd api && go mod tidy
    @ cd api && swag init
    @ cd api && swag fmt
    @ cd api && air

[group ('prod')]
build:
    @ cd api && go mod tidy
    @ cd api && go build -ldflags="-s -w -X main.versionTag=$(git rev-parse --short HEAD)" -trimpath -o tmp/api.exe ./cmd/api

[group('deploy')]
deploy-build:
    @ docker build \
    -t ghcr.io/safepolymed-uds/precisiondosing-api-go:latest .
run-deploy:
    @ docker run --rm -it --env-file api/.env -p 3333:3333 ghcr.io/safepolymed-uds/precisiondosing-api-go:latest

# Deletes feature branch after merging
[group('git')]
git-done branch=`git rev-parse --abbrev-ref HEAD`:
    @ git checkout main
    @ git diff --no-ext-diff --quiet --exit-code
    @ git pull --rebase github main
    @ git diff --no-ext-diff --quiet --exit-code {{branch}}
    @ git branch -D {{branch}}

# Updates all submodules
[group('git')]
git-update:
    @ git submodule update --remote --merge

# Installs Air and sets default .env
[group('init')]
init:
    @ go install github.com/air-verse/air@latest
    @ go install github.com/swaggo/swag/cmd/swag@latest
    @ scoop install main/golangci-lint
    @ cp api/cfg/default_env api/.env
