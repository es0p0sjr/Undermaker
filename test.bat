@echo off
cd /d "%~dp0"
set "GOCACHE=%~dp0.go-cache"
set "GOPROXY=off"
set "GOTOOLCHAIN=local"
go test ./...
if errorlevel 1 (
    pause
    exit /b 1
)
call npm.cmd test
set "TEST_RESULT=%ERRORLEVEL%"
pause
exit /b %TEST_RESULT%
