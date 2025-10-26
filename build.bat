@echo off
echo ATHENA Platform Build System

REM Create bin directory
if not exist bin mkdir bin

REM Update dependencies
echo Updating dependencies...
go mod download
go mod tidy

REM Build services
echo Building services...

echo Building api-gateway...
go build -o bin\api-gateway.exe .\cmd\api-gateway
if %errorlevel% equ 0 (
    echo ✓ Built api-gateway
) else (
    echo ✗ Failed to build api-gateway
)

echo Building template-service...
go build -o bin\template-service.exe .\cmd\template-service
if %errorlevel% equ 0 (
    echo ✓ Built template-service
) else (
    echo ✗ Failed to build template-service
)

echo Building nlp-service...
go build -o bin\nlp-service.exe .\cmd\nlp-service
if %errorlevel% equ 0 (
    echo ✓ Built nlp-service
) else (
    echo ✗ Failed to build nlp-service
)

echo Building provisioning-service...
go build -o bin\provisioning-service.exe .\cmd\provisioning-service
if %errorlevel% equ 0 (
    echo ✓ Built provisioning-service
) else (
    echo ✗ Failed to build provisioning-service
)

echo Building device-service...
go build -o bin\device-service.exe .\cmd\device-service
if %errorlevel% equ 0 (
    echo ✓ Built device-service
) else (
    echo ✗ Failed to build device-service
)

echo Building telemetry-service...
go build -o bin\telemetry-service.exe .\cmd\telemetry-service
if %errorlevel% equ 0 (
    echo ✓ Built telemetry-service
) else (
    echo ✗ Failed to build telemetry-service
)

echo Building ota-service...
go build -o bin\ota-service.exe .\cmd\ota-service
if %errorlevel% equ 0 (
    echo ✓ Built ota-service
) else (
    echo ✗ Failed to build ota-service
)

echo Building CLI...
go build -o bin\athena-cli.exe .\cmd\cli
if %errorlevel% equ 0 (
    echo ✓ Built athena-cli
) else (
    echo ✗ Failed to build athena-cli
)

echo.
echo Build complete!
echo.
echo Available binaries:
dir bin\*.exe /b