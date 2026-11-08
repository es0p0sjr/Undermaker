@echo off
cd /d "%~dp0"
if exist "dist\rework-studio.exe" (
  "dist\rework-studio.exe" --root "%~dp0." --open
) else (
  set "GOCACHE=%~dp0.go-cache"
  set "GOPROXY=off"
  set "GOTOOLCHAIN=local"
  go run ./ReworkStudio --root . --open
)
if errorlevel 1 pause
